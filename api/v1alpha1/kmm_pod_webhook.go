/*
Copyright 2025.
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

package v1alpha1

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// podlog is for logging in this package.
var podlog = logf.Log.WithName("pod-webhook")

// +kubebuilder:object:generate=false
// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
// KMMPodMutator is responsible for mutating pods to add drain protection annotations
// for KMM builder pods.
type KMMPodMutator struct {
	// Client client.Client
}

// +kubebuilder:webhook:verbs=create,path=/mutate-v1-kmm-builder-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,versions=v1,name=kmm-builder-pod-protection.fusion.storage.openshift.io,admissionReviewVersions=v1,sideEffects=none

var _ webhook.CustomDefaulter = &KMMPodMutator{}

func (r *KMMPodMutator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	// r.Client = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithDefaulter(r).
		// TODO check if this is needed, I had some issues with the path being some default value, not respecting the kubebuilder annotation above
		WithCustomPath("/mutate-v1-kmm-builder-pod").
		Complete()
}

func (r *KMMPodMutator) Default(ctx context.Context, obj runtime.Object) error {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected a Pod object but got %T", obj)
	}

	podlog.Info("Mutating KMM builder pod", "pod", pod.Name, "namespace", pod.Namespace)

	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	// // Add the drain protection annotation with 25 minutes (1500 seconds)
	// // This annotation is used by cluster-autoscaler and other draining mechanisms
	// pod.Annotations["cluster-autoscaler.kubernetes.io/safe-to-evict"] = "false"

	// // Add a custom annotation to track when this protection was added
	// pod.Annotations["fusion.storage.openshift.io/drain-protection-until"] = time.Now().Add(25 * time.Minute).Format(time.RFC3339)

	// // Add annotation to indicate this is a protected KMM builder pod
	// pod.Annotations["fusion.storage.openshift.io/kmm-builder-protected"] = "true"

	// podlog.Info("Added drain protection annotations to KMM builder pod",
	// 	"pod", pod.Name,
	// 	"namespace", pod.Namespace,
	// 	"protectionUntil", pod.Annotations["fusion.storage.openshift.io/drain-protection-until"])
	var terminationseconds int64 = 3000
	pod.Spec.TerminationGracePeriodSeconds = &terminationseconds
	pod.Annotations["example-mutating-admission-webhook"] = "foo"
	podlog.Info("Annotated Pod")

	return nil
}
