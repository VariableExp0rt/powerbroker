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

// PersonaParameters are the configurable fields of a Persona.
type PersonaParameters struct {
	Name string `json:"name"`

	// +crossplane:generate:reference:type=PermissionSet
	// +crossplane:generate:reference:extractor=reference.ExternalName()
	// +crossplane:generate:reference:refFieldName=PermissionSetRefs
	// +crossplane:generate:reference:selectorFieldName=PermissionSetRefSelector
	PermissionSets           []string         `json:"permissionSets,omitempty"`
	PermissionSetRefs        []xpv1.Reference `json:"permissionSetRefs,omitempty"`
	PermissionSetRefSelector *xpv1.Selector   `json:"permissionSetRefSelector,omitempty"`
}

// PersonaObservation are the observable fields of a Persona.
type PersonaObservation struct {
	NodeID string `json:"nodeId,omitempty"`
	Status string `json:"status,omitempty"`
}

// A PersonaSpec defines the desired state of a Persona.
type PersonaSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       PersonaParameters `json:"forProvider"`
}

// A PersonaStatus represents the observed state of a Persona.
type PersonaStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          PersonaObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Persona is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,neo4j}
type Persona struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PersonaSpec   `json:"spec"`
	Status PersonaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PersonaList contains a list of Persona
type PersonaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Persona `json:"items"`
}

// Persona type metadata.
var (
	PersonaKind             = reflect.TypeOf(Persona{}).Name()
	PersonaGroupKind        = schema.GroupKind{Group: Group, Kind: PersonaKind}.String()
	PersonaKindAPIVersion   = PersonaKind + "." + SchemeGroupVersion.String()
	PersonaGroupVersionKind = SchemeGroupVersion.WithKind(PersonaKind)
)

func init() {
	SchemeBuilder.Register(&Persona{}, &PersonaList{})
}
