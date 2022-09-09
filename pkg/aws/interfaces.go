package aws

import (
	awsclient "github.com/aws/aws-sdk-go/aws/client"
	"github.com/go-logr/logr"
)

// Session represents an AWS session
type Session interface {
	Session() awsclient.ConfigProvider
}

// ClusterScoper is the interface for a workload cluster scope
type ClusterScoper interface {
	logr.Logger
	Session

	// ARN returns the workload cluster assumed role to operate.
	ARN() string
	// ClusterName returns the AWS infrastructure cluster name.
	ClusterName() string
	// ClusterNamespace returns the AWS infrastructure cluster namespace.
	ClusterNamespace() string
	// Installation returns the installation name.
	Installation() string
	// Region returns the AWS infrastructure cluster object region.
	Region() string
}
