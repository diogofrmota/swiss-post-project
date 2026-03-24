// operator/internal/controller/appconfig_controller.go
// Reconciles AppConfig resources by creating or updating a managed Deployment.
package controller

import (
	"context"

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

	swisspostv1alpha1 "github.com/diogofrmota/custom-operator/internal/api/v1alpha1"
)

// AppConfigReconciler reconciles AppConfig objects.
type AppConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=swisspost.io,resources=appconfigs;appconfigs/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *AppConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// 1. Fetch the AppConfig
	ac := &swisspostv1alpha1.AppConfig{}
	if err := r.Get(ctx, req.NamespacedName, ac); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Build the desired Deployment and set AppConfig as its owner
	desired := r.buildDeployment(ac)
	if err := controllerutil.SetControllerReference(ac, desired, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// 3. Create or update the Deployment
	existing := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, existing)
	if errors.IsNotFound(err) {
		log.Info("Creating Deployment", "name", desired.Name)
		return ctrl.Result{}, r.Create(ctx, desired)
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	// Patch replicas, image and env if they drifted
	existing.Spec.Replicas = desired.Spec.Replicas
	existing.Spec.Template.Spec.Containers[0].Image = desired.Spec.Template.Spec.Containers[0].Image
	existing.Spec.Template.Spec.Containers[0].Env = desired.Spec.Template.Spec.Containers[0].Env
	if err := r.Update(ctx, existing); err != nil {
		return ctrl.Result{}, err
	}

	// 4. Sync status from the Deployment back to the AppConfig
	ac.Status.AvailableReplicas = existing.Status.AvailableReplicas
	if existing.Status.AvailableReplicas == *existing.Spec.Replicas {
		ac.Status.Ready = "True"
	} else {
		ac.Status.Ready = "False"
	}
	return ctrl.Result{}, r.Status().Update(ctx, ac)
}

func (r *AppConfigReconciler) buildDeployment(ac *swisspostv1alpha1.AppConfig) *appsv1.Deployment {
	labels := map[string]string{
		"app":                          ac.Name,
		"app.kubernetes.io/managed-by": "custom-operator",
	}

	envVars := make([]corev1.EnvVar, len(ac.Spec.Env))
	for i, e := range ac.Spec.Env {
		envVars[i] = corev1.EnvVar{Name: e.Name, Value: e.Value}
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: ac.Name, Namespace: ac.Namespace, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: &ac.Spec.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{RunAsNonRoot: boolPtr(true)},
					Containers: []corev1.Container{{
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
					}},
				},
			},
		},
	}
}

func (r *AppConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&swisspostv1alpha1.AppConfig{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func boolPtr(b bool) *bool { return &b }