/*
Copyright 2019 The Knative Authors

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

package containersource

import (
	"context"
	"errors"
	"fmt"
	"knative.dev/pkg/resolver"
	"strings"

	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1listers "k8s.io/client-go/listers/apps/v1"
	"knative.dev/pkg/controller"

	status "knative.dev/eventing/pkg/apis/duck"
	"knative.dev/eventing/pkg/apis/sources/v1alpha1"
	listers "knative.dev/eventing/pkg/client/listers/sources/v1alpha1"
	"knative.dev/eventing/pkg/logging"
	"knative.dev/eventing/pkg/reconciler"
	"knative.dev/eventing/pkg/reconciler/containersource/resources"
	pkgrec "knative.dev/pkg/reconciler"
)

const (
	// Name of the corev1.Events emitted from the reconciliation process
	sourceReconciled         = "ContainerSourceReconciled"
	sourceReadinessChanged   = "ContainerSourceReadinessChanged"
	sourceUpdateStatusFailed = "ContainerSourceUpdateStatusFailed"
)

type Reconciler struct {
	*Core
	*reconciler.Base

	// listers index properties about resources
	containerSourceLister listers.ContainerSourceLister
	deploymentLister      appsv1listers.DeploymentLister

	sinkResolver *resolver.URIResolver
}

// Check that our Reconciler implements controller.Reconciler
var _ controller.Reconciler = (*Reconciler)(nil)

func (r *Reconciler) ReconcileKind(ctx context.Context, source *v1alpha1.ContainerSource) pkgrec.Event {
	// No need to reconcile if the source has been marked for deletion.
	if source.DeletionTimestamp != nil {
		return nil
	}

	source.Status.ObservedGeneration = source.Generation
	source.Status.InitializeConditions()

	annotations := make(map[string]string)
	// Then wire through any annotations / labels from the Source
	if source.ObjectMeta.Annotations != nil {
		for k, v := range source.ObjectMeta.Annotations {
			annotations[k] = v
		}
	}
	labels := make(map[string]string)
	if source.ObjectMeta.Labels != nil {
		for k, v := range source.ObjectMeta.Labels {
			labels[k] = v
		}
	}

	args := resources.ContainerArguments{
		Source:             source,
		Name:               source.Name,
		Namespace:          source.Namespace,
		Template:           source.Spec.Template,
		Image:              source.Spec.DeprecatedImage,
		Args:               source.Spec.DeprecatedArgs,
		Env:                source.Spec.DeprecatedEnv,
		ServiceAccountName: source.Spec.DeprecatedServiceAccountName,
		Annotations:        annotations,
		Labels:             labels,
	}

	err := r.setSinkURIArg(ctx, source, &args)
	if err != nil {
		return pkgrec.NewReconcilerEvent(corev1.EventTypeWarning, "SetSinkURIFailed", "Failed to set Sink URI: %v", err)
	}

	ra, err := r.reconcileReceiveAdapter(ctx, source, args)
	if err != nil {
		return fmt.Errorf("reconciling receive adapter: %v", err)
	}

	if status.DeploymentIsAvailable(&ra.Status, false) {
		if !source.Status.IsDeployed() {
			source.Status.MarkDeployed()
			return pkgrec.NewReconcilerEvent(corev1.EventTypeNormal, "DeploymentReady", "Deployment %q has %d ready replicas", ra.Name, ra.Status.ReadyReplicas)
		}
	}
	return nil
}

// setSinkURIArg attempts to get the sink URI from the sink reference and
// set it in the source status. On failure, the source's Sink condition is
// updated to reflect the error.
// If an error is returned from this function, the caller should also record
// an Event containing the error string.
func (r *Reconciler) setSinkURIArg(ctx context.Context, source *v1alpha1.ContainerSource, args *resources.ContainerArguments) error {

	if uri, ok := sinkArg(source); ok {
		source.Status.MarkSink(uri)
		return nil
	}

	if source.Spec.Sink == nil {
		source.Status.MarkNoSink("Missing", "Sink missing from spec")
		return errors.New("sink missing from spec")
	}

	dest := source.Spec.Sink.DeepCopy()
	if dest.Ref != nil {
		// To call URIFromDestination(), dest.Ref must have a Namespace. If there is
		// no Namespace defined in dest.Ref, we will use the Namespace of the source
		// as the Namespace of dest.Ref.
		if dest.Ref.Namespace == "" {
			//TODO how does this work with deprecated fields
			dest.Ref.Namespace = source.GetNamespace()
		}
	} else if dest.DeprecatedName != "" && dest.DeprecatedNamespace == "" {
		// If Ref is nil and the deprecated ref is present, we need to check for
		// DeprecatedNamespace. This can be removed when DeprecatedNamespace is
		// removed.
		dest.DeprecatedNamespace = source.GetNamespace()
	}

	sinkURI, err := r.sinkResolver.URIFromDestination(*dest, source)
	if err != nil {
		source.Status.MarkNoSink("NotFound", `Couldn't get Sink URI from %+v`, dest)
		return fmt.Errorf("getting sink URI: %v", err)
	}
	if source.Spec.Sink.DeprecatedAPIVersion != "" &&
		source.Spec.Sink.DeprecatedKind != "" &&
		source.Spec.Sink.DeprecatedName != "" {
		source.Status.MarkSinkWarnRefDeprecated(sinkURI)
	} else {
		source.Status.MarkSink(sinkURI)
	}

	args.Sink = sinkURI

	return nil
}

func sinkArg(source *v1alpha1.ContainerSource) (string, bool) {
	var args []string

	if source.Spec.Template != nil {
		for _, c := range source.Spec.Template.Spec.Containers {
			args = append(args, c.Args...)
		}
	}

	args = append(args, source.Spec.DeprecatedArgs...)

	for _, a := range args {
		if strings.HasPrefix(a, "--sink=") {
			return strings.Replace(a, "--sink=", "", -1), true
		}
	}

	return "", false
}

func (r *Reconciler) reconcileReceiveAdapter(ctx context.Context, src *v1alpha1.ContainerSource, args resources.ContainerArguments) (*appsv1.Deployment, error) {
	expected := resources.MakeDeployment(args)

	ra, err := r.KubeClientSet.AppsV1().Deployments(src.Namespace).Get(expected.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		ra, err = r.KubeClientSet.AppsV1().Deployments(src.Namespace).Create(expected)
		if err != nil {
			r.markNotDeployedRecordEvent(src, corev1.EventTypeWarning, "DeploymentCreateFailed", "Could not create deployment: %v", err)
			return nil, fmt.Errorf("creating new deployment: %v", err)
		}
		r.markDeployingAndRecordEvent(src, corev1.EventTypeNormal, "DeploymentCreated", "Created deployment %q", ra.Name)
		return ra, nil
	} else if err != nil {
		r.markDeployingAndRecordEvent(src, corev1.EventTypeWarning, "DeploymentGetFailed", "Error getting deployment: %v", err)
		return nil, fmt.Errorf("getting deployment: %v", err)
	} else if !metav1.IsControlledBy(ra, src) {
		r.markDeployingAndRecordEvent(src, corev1.EventTypeWarning, "DeploymentNotOwned", "Deployment %q is not owned by this ContainerSource", ra.Name)
		return nil, fmt.Errorf("deployment %q is not owned by ContainerSource %q", ra.Name, src.Name)
	} else if r.podSpecChanged(ra.Spec.Template.Spec, expected.Spec.Template.Spec) {
		ra.Spec.Template.Spec = expected.Spec.Template.Spec
		ra, err = r.KubeClientSet.AppsV1().Deployments(src.Namespace).Update(ra)
		if err != nil {
			return ra, fmt.Errorf("updating deployment: %v", err)
		}
		return ra, nil
	} else {
		logging.FromContext(ctx).Debug("Reusing existing receive adapter", zap.Any("receiveAdapter", ra))
	}
	return ra, nil
}

func (r *Reconciler) podSpecChanged(oldPodSpec corev1.PodSpec, newPodSpec corev1.PodSpec) bool {
	// Since the Deployment spec has fields defaulted by the webhook, it won't
	// be equal to expected. Use DeepDerivative to compare only the fields that
	// are set in newPodSpec.
	if !equality.Semantic.DeepDerivative(newPodSpec, oldPodSpec) {
		return true
	}
	if len(oldPodSpec.Containers) != len(newPodSpec.Containers) {
		return true
	}
	for i := range newPodSpec.Containers {
		if !equality.Semantic.DeepEqual(newPodSpec.Containers[i].Env, oldPodSpec.Containers[i].Env) {
			return true
		}
	}
	return false
}

func (r *Reconciler) markDeployingAndRecordEvent(source *v1alpha1.ContainerSource, evType string, reason string, messageFmt string, args ...interface{}) {
	r.Core.Recorder.Eventf(source, evType, reason, messageFmt, args...)
	source.Status.MarkDeploying(reason, messageFmt, args...)
}

func (r *Reconciler) markNotDeployedRecordEvent(source *v1alpha1.ContainerSource, evType string, reason string, messageFmt string, args ...interface{}) {
	r.Core.Recorder.Eventf(source, evType, reason, messageFmt, args...)
	source.Status.MarkNotDeployed(reason, messageFmt, args...)
}
