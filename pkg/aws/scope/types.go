package scope

import (
	"github.com/giantswarm/aws-rolling-node-operator/pkg/aws"
)

// ASGScope is a scope for use with the ASG reconciling service in cluster
type ASGScope interface {
	aws.ClusterScoper
}
