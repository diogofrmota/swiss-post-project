// operator/internal/api/v1alpha1/groupversion_info.go
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is the group and version for this API.
	GroupVersion = schema.GroupVersion{Group: "swisspost.io", Version: "v1alpha1"}

	// SchemeBuilder adds the types in this group to a scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
