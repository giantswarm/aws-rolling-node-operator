/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/microerror"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/giantswarm/aws-rolling-node-operator/pkg/aws/scope"
	"github.com/giantswarm/aws-rolling-node-operator/pkg/key"
	"github.com/giantswarm/aws-rolling-node-operator/pkg/refresh"
)

// LegacyClusterReconciler reconciles a Giant Swarm AWSCluster object
type LegacyControlplaneReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	Installation string
	recorder     record.EventRecorder
}

// +kubebuilder:rbac:groups=infrastructure.giantswarm.io,resources=awscontrolplane,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.giantswarm.io,resources=awscontrolplane/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.giantswarm.io,resources=awscontrolplane/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *LegacyControlplaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error
	logger := r.Log.WithValues("namespace", req.Namespace, "controlplane", req.Name)

	cp := &infrastructurev1alpha3.AWSControlPlane{}
	if err := r.Get(ctx, req.NamespacedName, cp); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, microerror.Mask(err)
	}

	if !key.InstanceRefresh(cp) {
		logger.Info(
			fmt.Sprintf("AWSControlPlane CR do not have required annotation '%s', ignoring CR",
				key.InstanceRefreshAnnotation))
		return defaultRequeue(), nil
	}

	minHealthyPercentage, err := key.MinHealthyPercentage(cp)
	if err != nil {
		return defaultRequeue(), microerror.Mask(err)
	}

	clusterKey := types.NamespacedName{Name: key.Cluster(cp), Namespace: cp.GetNamespace()}
	cluster := &infrastructurev1alpha3.AWSCluster{}
	if err := r.Get(ctx, clusterKey, cluster); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, microerror.Mask(err)
	}

	accountID, arn, err := key.AWSAccountDetails(ctx, r.Client, cluster)
	if err != nil {
		return defaultRequeue(), microerror.Mask(err)
	}

	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		AccountID:        accountID,
		ARN:              arn,
		ClusterName:      cluster.Name,
		ClusterNamespace: cluster.Namespace,
		Installation:     r.Installation,
		Region:           cluster.Spec.Provider.Region,

		Logger: logger,
	})
	if err != nil {
		return reconcile.Result{}, microerror.Mask(err)
	}

	// Create InstanceRefresh service.
	instanceRefreshService := refresh.New(clusterScope, r.Client)
	startRefresh := make(chan bool)

	// ASG filter ControlPlane
	filter := map[string]string{
		key.ControlPlaneLabel: key.Controlplane(cp),
	}

	go func() {
		startEvent := <-startRefresh
		if startEvent {
			r.sendEvent(cp, v1.EventTypeNormal, "InstanceRefreshIsStarting", "Starting to replace all master nodes.")
		}
	}()

	err = instanceRefreshService.Refresh(ctx, minHealthyPercentage, filter, startRefresh)
	if _, ok := err.(awserr.Error); ok {
		return defaultRequeue(), microerror.Mask(err)
	} else if err != nil {
		r.sendEvent(cp, v1.EventTypeWarning, "InstanceRefreshCancelled", err.Error())
	} else {
		r.sendEvent(cp, v1.EventTypeNormal, "InstanceRefreshSuccessful", "Replaced all master nodes.")
	}

	if err := r.Get(ctx, req.NamespacedName, cp); err != nil {
		logger.Error(err, "ControlPlane does not exist")
		return ctrl.Result{}, microerror.Mask(err)
	}

	delete(cp.Annotations, key.InstanceRefreshAnnotation)
	delete(cp.Annotations, key.CancelInstanceRefreshAnnotation)
	err = r.Update(ctx, cp)
	if errors.IsConflict(err) {
		logger.Info("Failed to remove annotation on AWSControlPlane CR, conflict trying to update object")
	} else if err != nil {
		logger.Error(err, "failed to remove annotation on AWSControlPlane CR")
		return ctrl.Result{}, microerror.Mask(err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LegacyControlplaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("aws-controlplane-node-rolling-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha3.AWSControlPlane{}).
		Complete(r)
}

func (r *LegacyControlplaneReconciler) sendEvent(cp *v1alpha3.AWSControlPlane, eventtype, reason, message string) {
	r.recorder.Event(cp, eventtype, reason, message)
}
