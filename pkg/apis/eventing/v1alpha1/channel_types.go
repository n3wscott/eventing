/*
 * Copyright 2018 The Knative Authors
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
	"github.com/knative/pkg/apis/duck"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/knative/pkg/webhook"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Channel is an abstract resource that implements the Subscribable and Sinkable
// contracts. The Provisioner provisions infrastructure to accepts events and
// deliver to Subscriptions.
type Channel struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the Channel.
	Spec ChannelSpec `json:"spec,omitempty"`

	// Status represents the current state of the Channel. This data may be out of
	// date.
	// +optional
	Status ChannelStatus `json:"status,omitempty"`
}

// Check that Channel can be validated, can be defaulted, and has immutable fields.
var _ apis.Validatable = (*Channel)(nil)
var _ apis.Defaultable = (*Channel)(nil)
var _ apis.Immutable = (*Channel)(nil)
var _ runtime.Object = (*Channel)(nil)
var _ webhook.GenericCRD = (*Channel)(nil)

// Check that Channel implements the Conditions duck type.
var _ = duck.VerifyType(&Channel{}, &duckv1alpha1.Conditions{})
var _ = duck.VerifyType(&Channel{}, &duckv1alpha1.Channelable{})
var _ = duck.VerifyType(&Channel{}, &duckv1alpha1.Subscribable{})
var _ = duck.VerifyType(&Channel{}, &duckv1alpha1.Sinkable{})

// ChannelSpec specifies the Provisioner backing a channel and the configuration
// arguments for a Channel.
type ChannelSpec struct {
	// TODO: Generation does not work correctly with CRD. They are scrubbed
	// by the APIserver (https://github.com/kubernetes/kubernetes/issues/58778)
	// So, we add Generation here. Once that gets fixed, remove this and use
	// ObjectMeta.Generation instead.
	// +optional
	Generation int64 `json:"generation,omitempty"`

	// Provisioner defines the name of the Provisioner backing this channel.
	// TODO: +optional If missing, a default Provisioner may be selected for the Channel.
	Provisioner *ProvisionerReference `json:"provisioner,omitempty"`

	// Arguments defines the arguments to pass to the Provisioner which provisions
	// this Channel.
	// +optional
	Arguments *runtime.RawExtension `json:"arguments,omitempty"`

	// Subscribers is information about subscriptions used to implement message forwarding.
	Subscribers []ChannelSubscriberSpec `json:"subscribers,omitempty"`
}

// ChannelSubscriberSpec defines a single subscriber to a Channel.
type ChannelSubscriberSpec struct {
	// CallableDomain is the endpoint for the call
	// +OneOf(CallableDomain, SinkableDomain)
	// +optional
	CallableDomain string `json:"callableDomain,omitempty"`
	// SinkableDomain is the endpoint for the result
	// +OneOf(CallableDomain, SinkableDomain)
	// +optional
	SinkableDomain string `json:"sinkableDomain,omitempty"`
}

var chanCondSet = duckv1alpha1.NewLivingConditionSet(ChannelConditionProvisioned)

// ChannelStatus represents the current state of a Channel.
type ChannelStatus struct {
	// ObservedGeneration is the most recent generation observed for this Channel.
	// It corresponds to the Channel's generation, which is updated on mutation by
	// the API Server.
	// TODO: The above comment is only true once
	// https://github.com/kubernetes/kubernetes/issues/58778 is fixed.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Channel is Sinkable. It currently exposes the endpoint as top-level domain
	// that will distribute traffic over the provided targets from inside the cluster.
	// It generally has the form {channel}.{namespace}.svc.cluster.local
	Sinkable duckv1alpha1.Sinkable `json:"sinkable,omitempty"`

	// Channel is Subscribable. It just points to itself
	Subscribable duckv1alpha1.Subscribable `json:"subscribable,omitempty"`

	// Represents the latest available observations of a channel's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions duckv1alpha1.Conditions `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

const (
	// ChannelConditionReady has status True when the Channel is ready to accept
	// traffic.
	ChannelConditionReady = duckv1alpha1.ConditionReady

	// ChannelConditionProvisioned has status True when the Channel's backing
	// resources have been provisioned.
	ChannelConditionProvisioned duckv1alpha1.ConditionType = "Provisioned"
)

// GetCondition returns the condition currently associated with the given type, or nil.
func (cs *ChannelStatus) GetCondition(t duckv1alpha1.ConditionType) *duckv1alpha1.Condition {
	return chanCondSet.Manage(cs).GetCondition(t)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ChannelList is a collection of Channels.
type ChannelList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Channel `json:"items"`
}
