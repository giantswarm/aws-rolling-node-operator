package asg

import (
	"github.com/aws/aws-sdk-go/service/autoscaling/autoscalingiface"

	"github.com/giantswarm/aws-rolling-node-operator/pkg/aws/scope"
)

// Service holds a collection of interfaces.
type Service struct {
	scope  scope.ASGScope
	Client autoscalingiface.AutoScalingAPI
}

// NewService returns a new service given the S3 api client.
func NewService(clusterScope scope.ASGScope) *Service {
	return &Service{
		scope:  clusterScope,
		Client: scope.NewASGClient(clusterScope, clusterScope.ARN()),
	}
}
