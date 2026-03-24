// operator/internal/controller/appconfig_controller_test.go
package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	swisspostv1alpha1 "github.com/diogofrmota/custom-operator/internal/api/v1alpha1"
)

var _ = Describe("AppConfig Controller", func() {
	const (
		resourceName      = "test-appconfig"
		resourceNamespace = "default"
		timeout           = 10 * time.Second
		interval          = 250 * time.Millisecond
	)

	ctx := context.Background()
	key := types.NamespacedName{Name: resourceName, Namespace: resourceNamespace}

	AfterEach(func() {
		ac := &swisspostv1alpha1.AppConfig{}
		if err := k8sClient.Get(ctx, key, ac); err == nil {
			Expect(k8sClient.Delete(ctx, ac)).To(Succeed())
		}
	})

	Context("When creating an AppConfig", func() {
		It("Should create a managed Deployment", func() {
			By("Creating the AppConfig resource")
			ac := &swisspostv1alpha1.AppConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: resourceNamespace,
				},
				Spec: swisspostv1alpha1.AppConfigSpec{
					Image:    "nginx:alpine",
					Replicas: 2,
					Env: []swisspostv1alpha1.EnvVar{
						{Name: "ENV", Value: "test"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, ac)).To(Succeed())

			By("Checking that a Deployment is created")
			dep := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, key, dep)
			}, timeout, interval).Should(Succeed())

			Expect(*dep.Spec.Replicas).To(Equal(int32(2)))
			Expect(dep.Spec.Template.Spec.Containers[0].Image).To(Equal("nginx:alpine"))
			Expect(dep.Spec.Template.Spec.Containers[0].Env).To(ContainElement(
				corev1.EnvVar{Name: "ENV", Value: "test"},
			))
		})

		It("Should set the owner reference on the Deployment", func() {
			By("Creating the AppConfig resource")
			ac := &swisspostv1alpha1.AppConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: resourceNamespace,
				},
				Spec: swisspostv1alpha1.AppConfigSpec{
					Image:    "nginx:alpine",
					Replicas: 1,
				},
			}
			Expect(k8sClient.Create(ctx, ac)).To(Succeed())

			By("Verifying the owner reference")
			dep := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, key, dep)
			}, timeout, interval).Should(Succeed())

			Expect(dep.OwnerReferences).To(HaveLen(1))
			Expect(dep.OwnerReferences[0].Kind).To(Equal("AppConfig"))
			Expect(dep.OwnerReferences[0].Name).To(Equal(resourceName))
		})

		It("Should update the Deployment when the AppConfig spec changes", func() {
			By("Creating the AppConfig resource")
			ac := &swisspostv1alpha1.AppConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: resourceNamespace,
				},
				Spec: swisspostv1alpha1.AppConfigSpec{
					Image:    "nginx:1.25",
					Replicas: 1,
				},
			}
			Expect(k8sClient.Create(ctx, ac)).To(Succeed())

			dep := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, key, dep)
			}, timeout, interval).Should(Succeed())

			By("Updating the image in the AppConfig spec")
			Expect(k8sClient.Get(ctx, key, ac)).To(Succeed())
			ac.Spec.Image = "nginx:1.26"
			Expect(k8sClient.Update(ctx, ac)).To(Succeed())

			By("Checking that the Deployment image is updated")
			Eventually(func() string {
				if err := k8sClient.Get(ctx, key, dep); err != nil {
					return ""
				}
				return dep.Spec.Template.Spec.Containers[0].Image
			}, timeout, interval).Should(Equal("nginx:1.26"))
		})
	})
})


