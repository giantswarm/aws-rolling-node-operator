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
	"time"

	"github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	infrastructurev1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/microerror"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/giantswarm/aws-rolling-node-operator/pkg/aws/scope"
	"github.com/giantswarm/aws-rolling-node-operator/pkg/key"
	"github.com/giantswarm/aws-rolling-node-operator/pkg/refresh"
)

// LegacyClusterReconciler reconciles a Giant Swarm AWSCluster object
type LegacyClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	Installation string
	recorder     record.EventRecorder
}

// +kubebuilder:rbac:groups=infrastructure.giantswarm.io,resources=awscluster,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.giantswarm.io,resources=awscluster/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.giantswarm.io,resources=awscluster/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *LegacyClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error
	logger := r.Log.WithValues("namespace", req.Namespace, "cluster", req.Name)

	cluster := &infrastructurev1alpha3.AWSCluster{}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, microerror.Mask(err)
	}

	if !key.RefreshInstances(cluster) {
		logger.Info(
			fmt.Sprintf("AWSCluster CR do not have required annotation '%s', ignoring CR",
				key.RefreshInstancesAnnotation))
		return defaultRequeue(), nil
	}

	minHealthyPercentage, err := key.MinHealthyPercentage(cluster)
	if err != nil {
		return defaultRequeue(), microerror.Mask(err)
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

	err = instanceRefreshService.Reconcile(ctx, minHealthyPercentage, nil)
	if err != nil {
		return ctrl.Result{}, microerror.Mask(err)
	}

	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		logger.Error(err, "Cluster does not exist")
		return ctrl.Result{}, microerror.Mask(err)
	}

	delete(cluster.Annotations, key.RefreshInstancesAnnotation)
	err = r.Update(ctx, cluster)
	if errors.IsConflict(err) {
		logger.Info("Failed to remove annotation on AWSCluster CR, conflict trying to update object")
		return ctrl.Result{}, nil
	} else if err != nil {
		logger.Error(err, "failed to remove annotation on AWSCluster CR")
		return ctrl.Result{}, microerror.Mask(err)
	}
	r.sendEvent(cluster, v1.EventTypeNormal, "InstancesRefreshed", "Refreshed all master and worker instances.")

	return defaultRequeue(), nil
}

func (r *LegacyClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("aws-cluster-node-rolling-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1alpha3.AWSCluster{}).
		Complete(r)
}

func (r *LegacyClusterReconciler) sendEvent(cluster *v1alpha3.AWSCluster, eventtype, reason, message string) {
	r.recorder.Event(cluster, eventtype, reason, message)
}

func defaultRequeue() reconcile.Result {
	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: time.Minute * 5,
	}
}
