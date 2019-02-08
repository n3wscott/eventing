/*
 * Copyright 2019 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1alpha1

import (
	"github.com/knative/pkg/apis"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/knative/pkg/webhook"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Trigger struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the Trigger.
	Spec TriggerSpec `json:"spec,omitempty"`

	// Status represents the current state of the Trigger. This data may be out of
	// date.
	// +optional
	Status TriggerStatus `json:"status,omitempty"`
}

// Check that Trigger can be validated, can be defaulted, and has immutable fields.
var _ apis.Validatable = (*Trigger)(nil)
var _ apis.Defaultable = (*Trigger)(nil)
var _ apis.Immutable = (*Trigger)(nil)
var _ runtime.Object = (*Trigger)(nil)
var _ webhook.GenericCRD = (*Trigger)(nil)

type TriggerSpec struct {
	// TODO By enabling the status subresource metadata.generation should increment
	// thus making this property obsolete.
	//
	// We should be able to drop this property with a CRD conversion webhook
	// in the future
	//
	// +optional
	DeprecatedGeneration int64 `json:"generation,omitempty"`

	Broker string `json:"broker,omitempty"`

	// +optional
	Filter *FilterSelector `json:"filter,omitempty"`

	Subscriber *SubscriberSpec `json:"subscriber,omitempty"`
}

type FilterSelector struct {
	Headers map[string]string `json:"headers,omitempty" protobuf:"bytes,1,rep,name=headers"`
}

var triggerCondSet = duckv1alpha1.NewLivingConditionSet(TriggerConditionBrokerExists, TriggerConditionKubernetesService, TriggerConditionVirtualService, TriggerConditionSubscribed)

// TriggerStatus represents the current state of a Trigger.
type TriggerStatus struct {
	// ObservedGeneration is the most recent generation observed for this Trigger.
	// It corresponds to the Trigger's generation, which is updated on mutation by
	// the API Server.
	// TODO: The above comment is only true once
	// https://github.com/kubernetes/kubernetes/issues/58778 is fixed.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Represents the latest available observations of a trigger's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions duckv1alpha1.Conditions `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	SubscriberURI string `json:"subscriberURI,omitempty"`
}

const (
	TriggerConditionReady = duckv1alpha1.ConditionReady

	TriggerConditionBrokerExists duckv1alpha1.ConditionType = "BrokerExists"

	TriggerConditionKubernetesService duckv1alpha1.ConditionType = "KubernetesService"

	TriggerConditionVirtualService duckv1alpha1.ConditionType = "VirtualService"

	TriggerConditionSubscribed duckv1alpha1.ConditionType = "Subscribed"
)

// GetCondition returns the condition currently associated with the given type, or nil.
func (ts *TriggerStatus) GetCondition(t duckv1alpha1.ConditionType) *duckv1alpha1.Condition {
	return triggerCondSet.Manage(ts).GetCondition(t)
}

// IsReady returns true if the resource is ready overall.
func (ts *TriggerStatus) IsReady() bool {
	return triggerCondSet.Manage(ts).IsHappy()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (ts *TriggerStatus) InitializeConditions() {
	triggerCondSet.Manage(ts).InitializeConditions()
}

func (ts *TriggerStatus) MarkBrokerExists() {
	triggerCondSet.Manage(ts).MarkTrue(TriggerConditionBrokerExists)
}

func (ts *TriggerStatus) MarkBrokerDoesNotExists() {
	triggerCondSet.Manage(ts).MarkFalse(TriggerConditionBrokerExists, "doesNotExist", "Broker does not exist")
}

func (ts *TriggerStatus) MarkKubernetesServiceExists() {
	triggerCondSet.Manage(ts).MarkTrue(TriggerConditionKubernetesService)
}

func (ts *TriggerStatus) MarkVirtualServiceExists() {
	triggerCondSet.Manage(ts).MarkTrue(TriggerConditionVirtualService)
}

func (ts *TriggerStatus) MarkSubscribed() {
	triggerCondSet.Manage(ts).MarkTrue(TriggerConditionSubscribed)
}

func (ts *TriggerStatus) MarkNotSubscribed(reason, messageFormat string, messageA ...interface{}) {
	triggerCondSet.Manage(ts).MarkFalse(TriggerConditionSubscribed, reason, messageFormat, messageA...)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TriggerList is a collection of Triggers.
type TriggerList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Trigger `json:"items"`
}
