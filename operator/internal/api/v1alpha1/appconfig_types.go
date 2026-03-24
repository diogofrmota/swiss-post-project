// operator/internal/api/v1alpha1/appconfig_types.go
// Defines the Go types for the AppConfig custom resource.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AppConfigSpec defines the desired state of AppConfig.
type AppConfigSpec struct {
	// Image is the container image to deploy, e.g. ghcr.io/org/app:v1.2.3
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`

	// Replicas is the number of desired pod replicas.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	Replicas int32 `json:"replicas"`

	// Env holds optional environment variables injected into the container.
	// +optional
	Env []EnvVar `json:"env,omitempty"`
}

// EnvVar represents a single environment variable.
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// AppConfigStatus defines the observed state of AppConfig.
type AppConfigStatus struct {
	// Ready reports whether the managed Deployment is fully available.
	Ready string `json:"ready,omitempty"`

	// AvailableReplicas is the number of pods currently ready.
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// Conditions holds the latest available observations of the resource's state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=".spec.replicas"
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=".spec.image"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// AppConfig is the Schema for the appconfigs API.
type AppConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppConfigSpec   `json:"spec,omitempty"`
	Status AppConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AppConfigList contains a list of AppConfig.
type AppConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AppConfig{}, &AppConfigList{})
}
