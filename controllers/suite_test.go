/*
Copyright 2023.

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

package controllers

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	networkingv1alpha1 "github.com/snapp-incubator/lbaas-operator/api/v1alpha1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var externalService networkingv1alpha1.ExternalService
var manager ctrl.Manager
var cancel func()

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")

	ctx = ctrl.SetupSignalHandler()
	ctx, cancel = context.WithCancel(ctx)
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = networkingv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	manager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&ExternalServiceReconciler{
		Client: manager.GetClient(),
		Scheme: manager.GetScheme(),
	}).SetupWithManager(manager)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = manager.Start(ctx)
		Expect(err).NotTo(HaveOccurred(), "failed to run the manager")
	}()

	externalService = networkingv1alpha1.ExternalService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ExternalService",
			APIVersion: "networking.snappcloud.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-external-service",
			Namespace: "default",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: networkingv1alpha1.ExternalServiceSpec{
			Type:        networkingv1alpha1.StaticExternalServiceType,
			ServiceType: corev1.ServiceTypeClusterIP,
			Ports: []networkingv1alpha1.ExternalServicePort{
				{
					Name:     "test-port",
					Protocol: "TCP",
					Port:     8080,
				},
			},
			Static: networkingv1alpha1.Static{
				Addresses: []networkingv1alpha1.StaticEndpoint{
					{
						IP: "192.168.1.1",
					},
					{
						IP: "192.168.1.2",
					},
				},
			},
		},
	}
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("ExternalService controller", func() {
	Context("Skipping Reconcile an ExternalService", func() {
		var es networkingv1alpha1.ExternalService
		BeforeEach(func() {
			es = externalService
			es.Annotations = make(map[string]string)
			es.Annotations[SkipReconcileAnnotation] = "true"
			ctx := context.Background()
			err := k8sClient.Create(ctx, &es)
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			ctx := context.Background()
			err := k8sClient.Delete(ctx, &es)
			Expect(err).NotTo(HaveOccurred())
		})
		When("ExternalService created", func() {
			It("Should not create a Service object", func() {
				Eventually(func() bool {
					svc := &corev1.Service{}
					err := k8sClient.Get(context.Background(), types.NamespacedName{
						Namespace: "default",
						Name:      "test-external-service",
					}, svc)

					if err != nil {
						return true
					}

					return false
				}, time.Second, time.Millisecond*10).Should(BeTrue())
			})
			It("Should not create an Endpoints object", func() {
				Eventually(func() bool {
					ep := &corev1.Endpoints{}
					err := k8sClient.Get(context.Background(), types.NamespacedName{
						Namespace: "default",
						Name:      "test-external-service",
					}, ep)
					if err != nil {
						return true
					}

					return false
				}, time.Second, time.Millisecond*10).Should(BeTrue())
			})
			It("Should not update status", func() {
				Eventually(func() bool {
					var externalSvc networkingv1alpha1.ExternalService
					err := k8sClient.Get(context.Background(), types.NamespacedName{
						Namespace: "default",
						Name:      "test-external-service",
					}, &externalSvc)

					if err != nil {
						return false
					}

					if externalSvc.Status.ServiceAddress == "" {
						return true
					}
					return false
				}, time.Second, time.Millisecond*10).Should(BeTrue())
			})
		})
	})
	Context("Creating an ExternalService", func() {
		var es networkingv1alpha1.ExternalService
		BeforeEach(func() {
			es = externalService
			ctx := context.Background()
			err := k8sClient.Create(ctx, &es)
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			ctx := context.Background()
			err := k8sClient.Delete(ctx, &es)
			Expect(err).NotTo(HaveOccurred())
		})
		When("ExternalService created", func() {
			It("Should create a valid Service object", func() {
				Eventually(func() bool {
					svc := &corev1.Service{}
					err := k8sClient.Get(context.Background(), types.NamespacedName{
						Namespace: "default",
						Name:      "test-external-service",
					}, svc)
					if err != nil {
						return false
					}
					Expect(err).NotTo(HaveOccurred())
					Expect(svc.Name).To(Equal("test-external-service"))
					Expect(svc.Labels).To(Equal(externalService.Labels))

					return true
				}, time.Second, time.Millisecond*10).Should(BeTrue())
			})
			It("Should create a valid Endpoints object", func() {
				Eventually(func() bool {
					ep := &corev1.Endpoints{}
					err := k8sClient.Get(context.Background(), types.NamespacedName{
						Namespace: "default",
						Name:      "test-external-service",
					}, ep)
					if err != nil {
						return false
					}
					Expect(err).NotTo(HaveOccurred())
					Expect(ep.Name).To(Equal("test-external-service"))
					Expect(ep.Labels).To(Equal(externalService.Labels))
					Expect(ep.Subsets).To(ConsistOf(corev1.EndpointSubset{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "192.168.1.1",
							},
							{
								IP: "192.168.1.2",
							},
						},
						Ports: []corev1.EndpointPort{
							{
								Name:     "test-port",
								Port:     8080,
								Protocol: "TCP",
							},
						},
					}))

					return true
				}, time.Second, time.Millisecond*10).Should(BeTrue())
			})
			It("Should update status", func() {
				Eventually(func() bool {
					var externalSvc networkingv1alpha1.ExternalService
					err := k8sClient.Get(context.Background(), types.NamespacedName{
						Namespace: "default",
						Name:      "test-external-service",
					}, &externalSvc)

					if err != nil {
						return false
					}

					if externalSvc.Status.ServiceAddress == "test-external-service.default.svc.cluster.local" {
						return true
					}
					return false
				}, time.Second, time.Millisecond*10).Should(BeTrue())
			})
		})
	})
	Context("Updating an ExternalService", func() {
		var es networkingv1alpha1.ExternalService
		BeforeEach(func() {
			es = externalService
			ctx := context.Background()
			err := k8sClient.Create(ctx, &es)
			Expect(err).NotTo(HaveOccurred())

			// This line of code is for the purpose of waiting for the controller to complete necessary reconcile loops.
			time.Sleep(300 * time.Millisecond)

			var es2 networkingv1alpha1.ExternalService
			err = k8sClient.Get(ctx, types.NamespacedName{
				Namespace: "default",
				Name:      "test-external-service",
			}, &es2)

			Expect(err).NotTo(HaveOccurred())

			es2.Labels["test"] = "label"
			err = k8sClient.Update(ctx, &es2)
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			ctx := context.Background()
			err := k8sClient.Delete(ctx, &es)
			Expect(err).NotTo(HaveOccurred())
		})
		When("ExternalService updated", func() {
			It("Should update Service object", func() {
				Eventually(func() bool {
					svc := &corev1.Service{}
					err := k8sClient.Get(context.Background(), types.NamespacedName{
						Namespace: "default",
						Name:      "test-external-service",
					}, svc)
					if err != nil {
						return false
					}
					Expect(err).NotTo(HaveOccurred())
					Expect(svc.Name).To(Equal("test-external-service"))
					Expect(svc.Labels).To(Equal(externalService.Labels))

					return true
				}, time.Second, time.Millisecond*10).Should(BeTrue())
			})
			It("Should update Endpoints object", func() {
				Eventually(func() bool {
					ep := &corev1.Endpoints{}
					err := k8sClient.Get(context.Background(), types.NamespacedName{
						Namespace: "default",
						Name:      "test-external-service",
					}, ep)
					if err != nil {
						return false
					}
					Expect(err).NotTo(HaveOccurred())
					Expect(ep.Name).To(Equal("test-external-service"))
					Expect(ep.Labels).To(Equal(externalService.Labels))
					Expect(ep.Subsets).To(ConsistOf(corev1.EndpointSubset{
						Addresses: []corev1.EndpointAddress{
							{
								IP: "192.168.1.1",
							},
							{
								IP: "192.168.1.2",
							},
						},
						Ports: []corev1.EndpointPort{
							{
								Name:     "test-port",
								Port:     8080,
								Protocol: "TCP",
							},
						},
					}))

					return true
				}, time.Second, time.Millisecond*10).Should(BeTrue())
			})
		})
	})
	// EnvTest does not run the garbage collector when running the APIServer. So we can just check if the object has a valid OwnerReference
	// more info: https://github.com/kubernetes-sigs/controller-runtime/issues/626
	Context("Deleting an ExternalService", func() {
		var es networkingv1alpha1.ExternalService
		BeforeEach(func() {
			es = externalService
			ctx := context.Background()
			err := k8sClient.Create(ctx, &es)
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Delete(ctx, &es)
			Expect(err).NotTo(HaveOccurred())
		})
		When("ExternalService deleted", func() {
			It("Should delete Service object", func() {
				Eventually(func() bool {
					svc := &corev1.Service{}
					err := k8sClient.Get(context.Background(), types.NamespacedName{
						Namespace: "default",
						Name:      "test-external-service",
					}, svc)
					if err != nil {
						return false
					}

					Expect(svc.OwnerReferences[0].Name).To(Equal("test-external-service"))

					return true
				}, time.Second, time.Millisecond*10).Should(BeTrue())
			})
			It("Should delete Endpoints object", func() {
				Eventually(func() bool {
					ep := &corev1.Endpoints{}
					err := k8sClient.Get(context.Background(), types.NamespacedName{
						Namespace: "default",
						Name:      "test-external-service",
					}, ep)
					if err != nil {
						return false
					}
					Expect(ep.OwnerReferences[0].Name).To(Equal("test-external-service"))
					return true
				}, time.Second, time.Millisecond*10).Should(BeTrue())
			})
		})
	})
})
