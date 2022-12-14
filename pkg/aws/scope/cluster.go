package scope

import (
	"github.com/aws/aws-sdk-go/aws"
	awsclient "github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/klog/klogr"
)

// ClusterScopeParams defines the input parameters used to create a new Scope.
type ClusterScopeParams struct {
	AccountID        string
	ARN              string
	ClusterName      string
	ClusterNamespace string
	ConfigName       string
	Installation     string
	Region           string

	Logger  logr.Logger
	Session awsclient.ConfigProvider
}

// NewClusterScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewClusterScope(params ClusterScopeParams) (*ClusterScope, error) {
	if params.AccountID == "" {
		return nil, errors.New("failed to generate new scope from emtpy string AccountID")
	}
	if params.ARN == "" {
		return nil, errors.New("failed to generate new scope from emtpy string ARN")
	}
	if params.ClusterName == "" {
		return nil, errors.New("failed to generate new scope from emtpy string ClusterName")
	}
	if params.ClusterNamespace == "" {
		return nil, errors.New("failed to generate new scope from emtpy string ClusterNamespace")
	}
	if params.Installation == "" {
		return nil, errors.New("failed to generate new scope from emtpy string Installation")
	}
	if params.Region == "" {
		return nil, errors.New("failed to generate new scope from emtpy string Region")
	}

	if params.Logger == nil {
		params.Logger = klogr.New()
	}

	session, err := sessionForRegion(params.Region)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create aws session")
	}
	awsClientConfig := &aws.Config{Credentials: stscreds.NewCredentials(session, params.ARN)}

	stsClient := sts.New(session, awsClientConfig)
	_, err = stsClient.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get sts client")
	}

	return &ClusterScope{
		accountID:        params.AccountID,
		assumeRole:       params.ARN,
		clusterName:      params.ClusterName,
		clusterNamespace: params.ClusterNamespace,
		installation:     params.Installation,
		region:           params.Region,

		Logger:  params.Logger,
		session: session,
	}, nil
}

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	accountID        string
	assumeRole       string
	clusterName      string
	clusterNamespace string
	installation     string
	region           string

	logr.Logger
	session awsclient.ConfigProvider
}

// AccountID returns the account ID of the assumed role.
func (s *ClusterScope) AccountID() string {
	return s.accountID
}

// ARN returns the AWS SDK assumed role.
func (s *ClusterScope) ARN() string {
	return s.assumeRole
}

// ClusterName returns the name of AWS infrastructure cluster object.
func (s *ClusterScope) ClusterName() string {
	return s.clusterName
}

// ClusterNameSpace returns the namespace of AWS infrastructure cluster object.
func (s *ClusterScope) ClusterNamespace() string {
	return s.clusterNamespace
}

// Installation returns the name of the installation where the cluster object is located.
func (s *ClusterScope) Installation() string {
	return s.installation
}

// Region returns the region of the AWS infrastructure cluster object.
func (s *ClusterScope) Region() string {
	return s.region
}

// Session returns the AWS SDK session.
func (s *ClusterScope) Session() awsclient.ConfigProvider {
	return s.session
}
