package scope

import (
	awsclient "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"k8s.io/component-base/version"

	"github.com/giantswarm/aws-rolling-node-operator/pkg/aws"
)

// AWSClients contains all the aws clients used by the scopes
type AWSClients struct {
	ASG *autoscaling.AutoScaling
}

// NewASGClient creates a new ASG API client for a given session
func NewASGClient(session aws.Session, arn string) *autoscaling.AutoScaling {
	ASGClient := autoscaling.New(session.Session(), &awsclient.Config{Credentials: stscreds.NewCredentials(session.Session(), arn)})
	ASGClient.Handlers.Build.PushFrontNamed(getUserAgentHandler())

	return ASGClient
}

func getUserAgentHandler() request.NamedHandler {
	return request.NamedHandler{
		Name: "aws-rolling-node-operator/user-agent",
		Fn:   request.MakeAddToUserAgentHandler("awscluster", version.Get().String()),
	}
}
