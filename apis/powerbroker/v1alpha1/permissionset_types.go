/*
Copyright 2022 The Crossplane Authors.

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
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// AccountRoleBinding will become part of PermissionSetParameters
type AccountRoleBinding struct {
	Alias        string `json:"accountAlias,omitempty"`
	Account      string `json:"account"`
	AccountClass string `json:"accountClass,omitempty"`
	RoleName     string `json:"roleName"`
}

// TODO (VariableExp0rt): make account role binding general for each cloud provider
// that supports workload identity federation. Though specifically WI is not
// meant for this purpose, GCP and Azure support it, and the temporary nature
// of the credentials it vendors is beneficial for IAM activities

// PermissionSetParameters are the configurable fields of a PermissionSet.
type PermissionSetParameters struct {
	BindTo AccountRoleBinding `json:"bindTo"`
}

// PermissionSetObservation are the observable fields of a PermissionSet.
type PermissionSetObservation struct {
	NodeID string `json:"nodeId,omitempty"`
	Status string `json:"status,omitempty"`
}

// A PermissionSetSpec defines the desired state of a PermissionSet.
type PermissionSetSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       PermissionSetParameters `json:"forProvider"`
}

// A PermissionSetStatus represents the observed state of a PermissionSet.
type PermissionSetStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          PermissionSetObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A PermissionSet is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,neo4j}
type PermissionSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PermissionSetSpec   `json:"spec"`
	Status PermissionSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PermissionSetList contains a list of PermissionSet
type PermissionSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PermissionSet `json:"items"`
}

// PermissionSet type metadata.
var (
	PermissionSetKind             = reflect.TypeOf(PermissionSet{}).Name()
	PermissionSetGroupKind        = schema.GroupKind{Group: Group, Kind: PermissionSetKind}.String()
	PermissionSetKindAPIVersion   = PermissionSetKind + "." + SchemeGroupVersion.String()
	PermissionSetGroupVersionKind = SchemeGroupVersion.WithKind(PermissionSetKind)
)

func init() {
	SchemeBuilder.Register(&PermissionSet{}, &PermissionSetList{})
}
