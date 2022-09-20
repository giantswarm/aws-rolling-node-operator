package key

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClusterLabel           = "giantswarm.io/cluster"
	ControlPlaneLabel      = "giantswarm.io/control-plane"
	MachineDeploymentLabel = "giantswarm.io/machine-deployment"

	InstanceRefreshAnnotation       = "alpha.giantswarm.io/instance-refresh"
	CancelInstanceRefreshAnnotation = "alpha.giantswarm.io/cancel-instance-refresh"
	MinHealthyPercentageAnnotation  = "alpha.giantswarm.io/instance-refresh-min-healthy-percentage"
)

var (
	DefaultMinHealthyPercentage int64 = 90
)

func InstanceRefresh(getter AnnotationsGetter) bool {
	if _, ok := getter.GetAnnotations()[InstanceRefreshAnnotation]; !ok {
		return false
	}
	return true
}

func CancelInstanceRefresh(getter AnnotationsGetter) bool {
	if _, ok := getter.GetAnnotations()[CancelInstanceRefreshAnnotation]; !ok {
		return false
	}
	return true
}

func MinHealthyPercentage(getter AnnotationsGetter) (int64, error) {
	value, ok := getter.GetAnnotations()[MinHealthyPercentageAnnotation]
	if !ok {
		return DefaultMinHealthyPercentage, nil
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return DefaultMinHealthyPercentage, err
	}
	if v > 100 || v < 0 {
		return DefaultMinHealthyPercentage,
			fmt.Errorf("Minimum healthy percentage for refreshing instances must be between 0 and 100, got %v. Ignoring CR",
				v)
	}
	return int64(v), nil

}

func AWSAccountDetails(ctx context.Context, client client.Client, cluster *infrastructurev1alpha3.AWSCluster) (string, string, error) {
	// fetch ARN from the cluster to assume role for creating dependencies
	credentialName := cluster.Spec.Provider.CredentialSecret.Name
	credentialNamespace := cluster.Spec.Provider.CredentialSecret.Namespace
	var credentialSecret = &v1.Secret{}
	var credentialType = types.NamespacedName{Namespace: credentialNamespace, Name: credentialName}
	if err := client.Get(ctx, credentialType, credentialSecret); err != nil {
		return "", "", microerror.Mask(err)
	}

	secretByte, ok := credentialSecret.Data["aws.awsoperator.arn"]
	if !ok {
		return "",
			"",
			microerror.Mask(
				fmt.Errorf("Unable to extract ARN from secret %s for cluster %s",
					credentialName, cluster.Name))

	}

	// convert secret data secretByte into string
	arn := string(secretByte)

	// extract AccountID from ARN
	re := regexp.MustCompile(`[-]?\d[\d,]*[\.]?[\d{2}]*`)
	accountID := re.FindAllString(arn, 1)[0]

	if accountID == "" {
		return "",
			"",
			microerror.Mask(fmt.Errorf("Unable to extract Account ID from ARN %s", string(arn)))

	}
	return accountID, arn, nil
}

func Cluster(getter LabelsGetter) string {
	return getter.GetLabels()[ClusterLabel]
}

func Controlplane(getter LabelsGetter) string {
	return getter.GetLabels()[ControlPlaneLabel]
}

func MachineDeployment(getter LabelsGetter) string {
	return getter.GetLabels()[MachineDeploymentLabel]
}
