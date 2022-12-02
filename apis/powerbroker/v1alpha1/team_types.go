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

type ManagedByParameters struct {
	// +crossplane:generate:reference:type=User
	// +crossplane:generate:reference:extractor=github.com/crossplane/crossplane-runtime/pkg/reference.ExternalName()
	// +crossplane:generate:reference:refFieldName=UserRef
	// +crossplane:generate:reference:selectorFieldName=UserRefSelector
	User            string          `json:"user"`
	UserRef         *xpv1.Reference `json:"userRef,omitempty"`
	UserRefSelector *xpv1.Selector  `json:"userRefSelector,omitempty"`
}

// TeamParameters are the configurable fields of a Team.
type TeamParameters struct {
	Name      string              `json:"name"`
	ManagedBy ManagedByParameters `json:"managedBy"`

	// +crossplane:generate:reference:type=User
	// +crossplane:generate:reference:extractor=github.com/crossplane/crossplane-runtime/pkg/reference.ExternalName()
	// +crossplane:generate:reference:refFieldName=UserRefs
	// +crossplane:generate:reference:selectorFieldName=UserRefSelector
	Members         []string         `json:"members,omitempty"`
	UserRefs        []xpv1.Reference `json:"userRefs,omitempty"`
	UserRefSelector *xpv1.Selector   `json:"userSelector,omitempty"`

	// +crossplane:generate:reference:type=Persona
	// +crossplane:generate:reference:extractor=github.com/crossplane/crossplane-runtime/pkg/reference.ExternalName()
	// +crossplane:generate:reference:refFieldName=PersonaRefs
	// +crossplane:generate:reference:selectorFieldName=PersonaRefSelector
	Personas           []string         `json:"personas,omitempty"`
	PersonaRefs        []xpv1.Reference `json:"personaRefs,omitempty"`
	PersonaRefSelector *xpv1.Selector   `json:"personaSelector,omitempty"`
}

// TeamObservation are the observable fields of a Team.
type TeamObservation struct {
	NodeID string `json:"nodeId,omitempty"`
	Status string `json:"status,omitempty"`
}

// A TeamSpec defines the desired state of a Team.
type TeamSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       TeamParameters `json:"forProvider"`
}

// A TeamStatus represents the observed state of a Team.
type TeamStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          TeamObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A Team is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="MANAGED BY", type="string",JSONPath=".spec.forProvider.managedBy.userRef"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,neo4j}
type Team struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamSpec   `json:"spec"`
	Status TeamStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TeamList contains a list of Team
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Team `json:"items"`
}

// Team type metadata.
var (
	TeamKind             = reflect.TypeOf(Team{}).Name()
	TeamGroupKind        = schema.GroupKind{Group: Group, Kind: TeamKind}.String()
	TeamKindAPIVersion   = TeamKind + "." + SchemeGroupVersion.String()
	TeamGroupVersionKind = SchemeGroupVersion.WithKind(TeamKind)
)

func init() {
	SchemeBuilder.Register(&Team{}, &TeamList{})
}
