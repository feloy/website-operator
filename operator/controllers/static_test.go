package controllers

import (
	"context"
	"time"

	"example.com/website/v1alpha1/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Static controller", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Second * 1
	)

	var (
		ctx = context.Background()

		key = types.NamespacedName{
			Name:      "my-static",
			Namespace: "my-ns",
		}

		deploymentKey = types.NamespacedName{
			Name:      "my-static-deployment",
			Namespace: "my-ns",
		}

		serviceKey = types.NamespacedName{
			Name:      "my-static-service",
			Namespace: "my-ns",
		}

		hpaKey = types.NamespacedName{
			Name:      "my-static-hpa",
			Namespace: "my-ns",
		}
	)

	When("a Static resource is created", func() {

		var (
			created                v1alpha1.Static
			expectedOwnerReference metav1.OwnerReference
		)

		BeforeEach(func() {
			created = v1alpha1.Static{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: v1alpha1.StaticSpec{
					DiskSize:    *resource.NewQuantity(1024*1024, resource.BinarySI),
					Source:      "gs://my-bucket/",
					MinReplicas: 2,
					MaxReplicas: 4,
				},
			}

			Expect(k8sClient.Create(ctx, &created)).Should(Succeed())

			expectedOwnerReference = metav1.OwnerReference{
				Kind:               "Static",
				APIVersion:         "website.example.com/v1alpha1",
				Name:               "my-static",
				UID:                created.UID,
				Controller:         func(v bool) *bool { return &v }(true),
				BlockOwnerDeletion: func(v bool) *bool { return &v }(true),
			}
		})

		AfterEach(func() {
			k8sClient.Delete(ctx, &created)
		})

		Specify("a static website is created", func() {

			By("creating a deployment owned by the Static resource", func() {
				var deployment appsv1.Deployment
				Eventually(func() error {
					return k8sClient.Get(ctx, deploymentKey, &deployment)
				}, timeout, interval).Should(BeNil())
				Expect(deployment.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference))
			})

			By("creating a service owned by the Static resource", func() {
				var service v1.Service
				Eventually(func() error {
					return k8sClient.Get(ctx, serviceKey, &service)
				}, timeout, interval).Should(BeNil())
				Expect(service.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference))
			})

			By("creating an hpa owned by the Static resource", func() {
				var hpa autoscalingv1.HorizontalPodAutoscaler
				Eventually(func() error {
					return k8sClient.Get(ctx, hpaKey, &hpa)
				}, timeout, interval).Should(BeNil())
				Expect(hpa.ObjectMeta.OwnerReferences).To(ContainElement(expectedOwnerReference))
			})
		})

		When("the website is created", func() {
			var (
				deployment appsv1.Deployment
				service    v1.Service
				hpa        autoscalingv1.HorizontalPodAutoscaler
			)

			BeforeEach(func() {
				Eventually(func() error {
					return k8sClient.Get(ctx, deploymentKey, &deployment)
				}, timeout, interval).Should(BeNil())

				Eventually(func() error {
					return k8sClient.Get(ctx, serviceKey, &service)
				}, timeout, interval).Should(BeNil())

				Eventually(func() error {
					return k8sClient.Get(ctx, hpaKey, &hpa)
				}, timeout, interval).Should(BeNil())
			})

			When("the replicas count of the deployment changes", func() {

				const newReplicasCount = 3

				BeforeEach(func() {
					deployment.Status.Replicas = newReplicasCount
					Expect(k8sClient.Status().Update(ctx, &deployment)).To(Succeed())
				})

				Specify("the replicas value in the Static status changes", func() {
					Eventually(func() bool {
						f := &v1alpha1.Static{}
						if err := k8sClient.Get(ctx, key, f); err != nil {
							return false
						}
						return f.Status.Replicas == newReplicasCount
					}, timeout, interval).Should(BeTrue())
				})
			})

			When("the external IP of the service changes", func() {

				const newExternalIP = "10.0.0.10"

				BeforeEach(func() {
					service.Status.LoadBalancer = v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{
								IP: newExternalIP,
							},
						},
					}
					Expect(k8sClient.Status().Update(ctx, &service)).To(Succeed())
				})

				Specify("the ExternalIP value in the Static status changes", func() {
					Eventually(func() bool {
						f := &v1alpha1.Static{}
						if err := k8sClient.Get(ctx, key, f); err != nil {
							return false
						}
						return f.Status.ExternalIP == newExternalIP
					}, timeout, interval).Should(BeTrue())

				})
			})

			Context("the deployment strategy is RollingUpdate", func() {

				const initialStrategy = appsv1.RollingUpdateDeploymentStrategyType

				BeforeEach(func() {
					Expect(deployment.Spec.Strategy.Type).To(Equal(initialStrategy))
				})

				When("the deployment strategy is changed to Recreate", func() {

					const newStrategy = appsv1.RecreateDeploymentStrategyType

					BeforeEach(func() {
						deployment.Spec.Strategy = appsv1.DeploymentStrategy{
							Type: newStrategy,
						}
						Expect(k8sClient.Update(ctx, &deployment)).To(Succeed())
						Eventually(func() bool {
							f := &appsv1.Deployment{}
							if err := k8sClient.Get(ctx, deploymentKey, f); err != nil {
								return false
							}
							return f.Spec.Strategy.Type == newStrategy
						}, timeout, interval).Should(BeTrue())
					})

					It("is changed back to the original strategy", func() {
						Eventually(func() bool {
							f := &appsv1.Deployment{}
							if err := k8sClient.Get(ctx, deploymentKey, f); err != nil {
								return false
							}
							return f.Spec.Strategy.Type == initialStrategy
						}, timeout, interval).Should(BeTrue())
					})
				})
			})

			Context("the service port is 80", func() {

				const initialPort int32 = 80

				BeforeEach(func() {
					Expect(service.Spec.Ports[0].Port).To(Equal(initialPort))
				})

				When("the service port is changed to 8080", func() {

					const newPort int32 = 8080

					BeforeEach(func() {
						service.Spec.Ports[0].Port = newPort
						Expect(k8sClient.Update(ctx, &service)).To(Succeed())
						Eventually(func() bool {
							f := &v1.Service{}
							if err := k8sClient.Get(ctx, serviceKey, f); err != nil {
								return false
							}
							return f.Spec.Ports[0].Port == newPort
						}, timeout, interval).Should(BeTrue())
					})

					It("is changed back to the original port", func() {
						Eventually(func() bool {
							f := &v1.Service{}
							if err := k8sClient.Get(ctx, serviceKey, f); err != nil {
								return false
							}
							return f.Spec.Ports[0].Port == initialPort
						}, timeout, interval).Should(BeTrue())
					})
				})
			})

			Context("the hpa minReplicas equals the Static resource minReplicas", func() {

				var initialMinReplicas int32 = 2

				BeforeEach(func() {
					Expect(*hpa.Spec.MinReplicas).To(Equal(initialMinReplicas))
				})

				When("the hpa minReplicas is changed", func() {

					var newMinReplicas int32 = 1

					BeforeEach(func() {
						hpa.Spec.MinReplicas = &newMinReplicas
						Expect(k8sClient.Update(ctx, &hpa)).To(Succeed())
						Eventually(func() bool {
							f := &autoscalingv1.HorizontalPodAutoscaler{}
							if err := k8sClient.Get(ctx, hpaKey, f); err != nil {
								return false
							}
							return *f.Spec.MinReplicas == newMinReplicas
						}, timeout, interval).Should(BeTrue())
					})

					It("is changed back to the original minReplicas", func() {
						Eventually(func() bool {
							f := &autoscalingv1.HorizontalPodAutoscaler{}
							if err := k8sClient.Get(ctx, hpaKey, f); err != nil {
								return false
							}
							return *f.Spec.MinReplicas == initialMinReplicas
						}, timeout, interval).Should(BeTrue())
					})
				})
			})
		})
	})
})
