package refresh

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"

	"github.com/cenkalti/backoff/v4"

	"github.com/giantswarm/aws-rolling-node-operator/pkg/aws/scope"
	"github.com/giantswarm/aws-rolling-node-operator/pkg/aws/services/asg"
	"github.com/giantswarm/aws-rolling-node-operator/pkg/key"
)

type InstanceRefreshService struct {
	Client client.Client
	Scope  *scope.ClusterScope

	ASG *asg.Service
}

func New(scope *scope.ClusterScope, client client.Client) *InstanceRefreshService {
	return &InstanceRefreshService{
		Scope:  scope,
		Client: client,

		ASG: asg.NewService(scope),
	}
}

func (s *InstanceRefreshService) Refresh(ctx context.Context, minHealhtyPercentage int64, asgFilter map[string]string) error {
	asgInput := &autoscaling.DescribeAutoScalingGroupsInput{
		// default filter for ASGs
		Filters: []*autoscaling.Filter{
			{
				Name:   aws.String("tag-key"),
				Values: []*string{aws.String("giantswarm.io/cluster")},
			},
			{
				Name:   aws.String("tag-value"),
				Values: []*string{aws.String(s.Scope.ClusterName())},
			},
		},
	}

	// addtional filter for ASG, depending what ASG you wanna roll specifically (certain nodepools or controlplanes)
	for k, v := range asgFilter {
		filter := []*autoscaling.Filter{
			{
				Name:   aws.String("tag-key"),
				Values: []*string{aws.String(k)},
			},
			{
				Name:   aws.String("tag-value"),
				Values: []*string{aws.String(v)},
			},
		}
		asgInput.Filters = append(asgInput.Filters, filter...)
	}

	asgOutput, err := s.ASG.Client.DescribeAutoScalingGroups(asgInput)
	if err != nil {
		return err
	}

	for _, asg := range asgOutput.AutoScalingGroups {
		refreshStatus := &autoscaling.DescribeInstanceRefreshesInput{
			AutoScalingGroupName: asg.AutoScalingGroupName,
		}

		output, err := s.ASG.Client.DescribeInstanceRefreshes(refreshStatus)
		if err != nil {
			s.Scope.Logger.Error(err, "failed to describe instance refreshes")
			return err

		}
		if len(output.InstanceRefreshes) > 0 {
			if output.InstanceRefreshes[0].EndTime != nil {
				if !output.InstanceRefreshes[0].EndTime.UTC().Before(time.Now().UTC().Add(-30 * time.Minute)) {
					s.Scope.Logger.Info(
						fmt.Sprintf("ASG %s already refreshed within the last 30 minutes, skipping...",
							*asg.AutoScalingGroupName))
					continue
				}
			}
		}

		refreshInput := &autoscaling.StartInstanceRefreshInput{
			AutoScalingGroupName: asg.AutoScalingGroupName,
			DesiredConfiguration: &autoscaling.DesiredConfiguration{
				LaunchTemplate: &autoscaling.LaunchTemplateSpecification{
					LaunchTemplateId: asg.Instances[0].LaunchTemplate.LaunchTemplateId,
					Version:          aws.String("$Latest"),
				},
			},
			Preferences: &autoscaling.RefreshPreferences{
				CheckpointDelay:       nil,
				CheckpointPercentages: []*int64{},
				InstanceWarmup:        nil,
				MinHealthyPercentage:  aws.Int64(minHealhtyPercentage),
				SkipMatching:          nil,
			},
			Strategy: aws.String("Rolling"),
		}
		_, err = s.ASG.Client.StartInstanceRefresh(refreshInput)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case autoscaling.ErrCodeInstanceRefreshInProgressFault:
				s.Scope.Logger.Info(fmt.Sprintf("An instance refresh is already in progress for ASG %s.",
					*asg.AutoScalingGroupName))
			}
		} else if err != nil {
			s.Scope.Logger.Error(err, "failed to start instance refresh")
		}

		b := backoff.NewConstantBackOff(60 * time.Second)

		waitonRefresh := func() error {
			output, err := s.ASG.Client.DescribeInstanceRefreshes(refreshStatus)
			if err != nil {
				s.Scope.Logger.Error(err, "failed to describe instance refreshes")
				return backoff.Permanent(err)
			}
			if *output.InstanceRefreshes[0].Status == autoscaling.InstanceRefreshStatusSuccessful {
				s.Scope.Logger.Info(fmt.Sprintf("Successfully refreshed all instances in ASG %s",
					*asg.AutoScalingGroupName))
				return nil
			}

			if *output.InstanceRefreshes[0].Status == autoscaling.InstanceRefreshStatusCancelling {
				s.Scope.Logger.Info(fmt.Sprintf("Cancelling refreshing instances in ASG %s",
					*asg.AutoScalingGroupName))
				return nil
			}

			if *output.InstanceRefreshes[0].Status == autoscaling.InstanceRefreshStatusCancelled {
				s.Scope.Logger.Info(fmt.Sprintf("Cancelled refreshing instances in ASG %s",
					*asg.AutoScalingGroupName))
				return nil
			}

			s.Scope.Logger.Info(fmt.Sprintf("Refreshing instances in ASG %s, Status: %s",
				*asg.AutoScalingGroupName,
				*output.InstanceRefreshes[0].Status))

			if s.shouldCancel(ctx, asgFilter) {
				cancelInput := &autoscaling.CancelInstanceRefreshInput{
					AutoScalingGroupName: aws.String(*asg.AutoScalingGroupName),
				}
				_, err := s.ASG.Client.CancelInstanceRefresh(cancelInput)
				if err != nil {
					s.Scope.Logger.Error(err, "failed to cancel instance refresh")
					return nil
				}
				return backoff.Permanent(fmt.Errorf("cancelled instance refresh for ASG %s", *asg.AutoScalingGroupName))
			}

			return fmt.Errorf("ASG %s is not ready yet", *asg.AutoScalingGroupName)
		}
		err = backoff.Retry(waitonRefresh, b)
		if err != nil {
			s.Scope.Logger.Error(err, "refreshing instances failed")
			return err
		}
	}
	return nil
}

func (s *InstanceRefreshService) shouldCancel(ctx context.Context, asgFilter map[string]string) bool {
	if asgFilter != nil {
		if v, ok := asgFilter[key.ControlPlaneLabel]; ok {
			cp := &infrastructurev1alpha3.AWSControlPlane{}
			err := s.Client.Get(ctx, types.NamespacedName{Name: v, Namespace: s.Scope.ClusterNamespace()}, cp)
			if err != nil {
				s.Scope.Logger.Error(err, "failed to get AWSControlplane")
				return false
			}
			return key.CancelRefreshInstances(cp)
		}
		if v, ok := asgFilter[key.MachineDeploymentLabel]; ok {
			md := &infrastructurev1alpha3.AWSMachineDeployment{}
			err := s.Client.Get(ctx, types.NamespacedName{Name: v, Namespace: s.Scope.ClusterNamespace()}, md)
			if err != nil {
				s.Scope.Logger.Error(err, "failed to get AWSMachineDeployment")
				return false
			}
			return key.CancelRefreshInstances(md)
		}
	} else {
		cluster := &infrastructurev1alpha3.AWSCluster{}
		err := s.Client.Get(ctx, types.NamespacedName{Name: s.Scope.ClusterName(), Namespace: s.Scope.ClusterNamespace()}, cluster)
		if err != nil {
			s.Scope.Logger.Error(err, "failed to get AWSCluster")
			return false
		}
		return key.CancelRefreshInstances(cluster)
	}
	return false
}
