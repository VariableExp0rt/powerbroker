package apis

import (
	neo4jv1alpha1 "github.com/VariableExp0rt/powerbroker/apis/powerbroker/v1alpha1"
	v1alpha1 "github.com/VariableExp0rt/powerbroker/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		neo4jv1alpha1.SchemeBuilder.AddToScheme,
		v1alpha1.SchemeBuilder.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
