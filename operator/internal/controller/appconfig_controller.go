// operator/internal/controller/appconfig_controller.go
// Reconciles AppConfig resources by creating or updating a managed Deployment.
package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	swisspostv1alpha1 "github.com/<YOUR_USERNAME>/custom-operator/internal/api/v1alpha1"
)

const finalizerName = "swisspost.io/appconfig-finalizer"

// AppConfigReconciler reconciles AppConfig objects.
type AppConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=swisspost.io,resources=appconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=swisspost.io,resources=appconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=swisspost.io,resources=appconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (r *AppConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the AppConfig resource
	appConfig := &swisspostv1alpha1.AppConfig{}
	if err := r.Get(ctx, req.NamespacedName, appConfig); err != nil {
		if errors.IsNotFound(err) {
			// Resource deleted — nothing to do
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get AppConfig: %w", err)
	}

	// Handle deletion via finalizer
	if !appConfig.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, appConfig)
	}

	// Register finalizer if absent
	if !controllerutil.ContainsFinalizer(appConfig, finalizerName) {
		controllerutil.AddFinalizer(appConfig, finalizerName)
		if err := r.Update(ctx, appConfig); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile the managed Deployment
	if err := r.reconcileDeployment(ctx, appConfig); err != nil {
		logger.Error(err, "Failed to reconcile Deployment")
		return ctrl.Result{}, err
	}

	// Update status
	if err := r.updateStatus(ctx, appConfig); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Reconciliation complete", "name", req.Name, "namespace", req.Namespace)
	return ctrl.Result{}, nil
}

// reconcileDeployment creates or updates the Deployment owned by this AppConfig.
func (r *AppConfigReconciler) reconcileDeployment(ctx context.Context, ac *swisspostv1alpha1.AppConfig) error {
	desired := r.buildDeployment(ac)

	// Set AppConfig as owner so the Deployment is garbage-collected on deletion
	if err := controllerutil.SetControllerReference(ac, desired, r.Scheme); err != nil {
		return err
	}

	existing := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Update image and replicas if they drifted
	existing.Spec.Replicas = desired.Spec.Replicas
	existing.Spec.Template.Spec.Containers[0].Image = desired.Spec.Template.Spec.Containers[0].Image
	existing.Spec.Template.Spec.Containers[0].Env = desired.Spec.Template.Spec.Containers[0].Env
	return r.Update(ctx, existing)
}

// buildDeployment constructs the desired Deployment from the AppConfig spec.
func (r *AppConfigReconciler) buildDeployment(ac *swisspostv1alpha1.AppConfig) *appsv1.Deployment {
	labels := map[string]string{
		"app":                          ac.Name,
		"app.kubernetes.io/managed-by": "custom-operator",
	}

	// Convert EnvVar slice to corev1.EnvVar
	envVars := make([]corev1.EnvVar, len(ac.Spec.Env))
	for i, e := range ac.Spec.Env {
		envVars[i] = corev1.EnvVar{Name: e.Name, Value: e.Value}
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Name,
			Namespace: ac.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &ac.Spec.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: boolPtr(true),
					},
					Containers: []corev1.Container{
						{
							Name:            ac.Name,
							Image:           ac.Spec.Image,
							ImagePullPolicy: corev1.PullAlways,
							Env:             envVars,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("50m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
								ReadOnlyRootFilesystem:   boolPtr(true),
							},
						},
					},
				},
			},
		},
	}
}

// updateStatus syncs the AppConfig status with the observed Deployment state.
func (r *AppConfigReconciler) updateStatus(ctx context.Context, ac *swisspostv1alpha1.AppConfig) error {
	dep := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: ac.Name, Namespace: ac.Namespace}, dep); err != nil {
		return err
	}

	ac.Status.AvailableReplicas = dep.Status.AvailableReplicas
	if dep.Status.AvailableReplicas == *dep.Spec.Replicas {
		ac.Status.Ready = "True"
	} else {
		ac.Status.Ready = "False"
	}

	return r.Status().Update(ctx, ac)
}

// handleDeletion removes the finalizer once cleanup is done.
func (r *AppConfigReconciler) handleDeletion(ctx context.Context, ac *swisspostv1alpha1.AppConfig) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(ac, finalizerName) {
		// No external resources to clean up — just remove the finalizer
		controllerutil.RemoveFinalizer(ac, finalizerName)
		if err := r.Update(ctx, ac); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager registers the controller with the manager.
func (r *AppConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&swisspostv1alpha1.AppConfig{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func boolPtr(b bool) *bool { return &b }
