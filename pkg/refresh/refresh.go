package refresh

import (
	"context"
	"fmt"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"

	"github.com/cenkalti/backoff/v4"
	"github.com/giantswarm/aws-rolling-node-operator/pkg/aws/scope"
	"github.com/giantswarm/aws-rolling-node-operator/pkg/aws/services/asg"
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

func (s *InstanceRefreshService) Reconcile(ctx context.Context, minHealhtyPercentage int64, asgFilter map[string]string) error {
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
					*output.InstanceRefreshes[0].AutoScalingGroupName))
				return nil
			}

			if *output.InstanceRefreshes[0].Status == autoscaling.InstanceRefreshStatusCancelling {
				s.Scope.Logger.Info(fmt.Sprintf("Cancelling refreshing instances in ASG %s",
					*output.InstanceRefreshes[0].AutoScalingGroupName))
				return nil
			}

			if *output.InstanceRefreshes[0].Status == autoscaling.InstanceRefreshStatusCancelled {
				s.Scope.Logger.Info(fmt.Sprintf("Cancelled refreshing instances in ASG %s",
					*output.InstanceRefreshes[0].AutoScalingGroupName))
				return nil
			}

			s.Scope.Logger.Info(fmt.Sprintf("Refreshing instances in ASG %s, Status: %s",
				*output.InstanceRefreshes[0].AutoScalingGroupName,
				*output.InstanceRefreshes[0].Status))

			return fmt.Errorf("ASG %s is not ready yet", *output.InstanceRefreshes[0].AutoScalingGroupName)
		}
		err = backoff.Retry(waitonRefresh, b)
		if err != nil {
			s.Scope.Logger.Error(err, "refreshing instances failed")
			return err
		}
	}
	return nil
}
