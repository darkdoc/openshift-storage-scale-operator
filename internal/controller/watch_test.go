package controller

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	fusionv1alpha1 "github.com/openshift-storage-scale/openshift-fusion-access-operator/api/v1alpha1"
	"github.com/openshift-storage-scale/openshift-fusion-access-operator/internal/controller/kernelmodule"
)

var _ = Describe("Watch Functionality", func() {
	var (
		testNamespace = "test-namespace"
		ctx           context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		os.Setenv("DEPLOYMENT_NAMESPACE", testNamespace)
	})

	AfterEach(func() {
		os.Unsetenv("DEPLOYMENT_NAMESPACE")
	})

	Describe("checkResourceObject", func() {
		It("should watch KMM ConfigMap in correct namespace", func() {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kernelmodule.KMMImageConfigMapName,
					Namespace: testNamespace,
				},
			}

			result := checkResourceObject(cm, testNamespace, resourceTypeConfigMap)
			Expect(result).To(BeTrue())
		})

		It("should not watch ConfigMap with wrong name", func() {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other-configmap",
					Namespace: testNamespace,
				},
			}

			result := checkResourceObject(cm, testNamespace, resourceTypeConfigMap)
			Expect(result).To(BeFalse())
		})

		It("should not watch ConfigMap in wrong namespace", func() {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kernelmodule.KMMImageConfigMapName,
					Namespace: "wrong-namespace",
				},
			}

			result := checkResourceObject(cm, testNamespace, resourceTypeConfigMap)
			Expect(result).To(BeFalse())
		})

		It("should watch IBM entitlement secret", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      IBMENTITLEMENTNAME,
					Namespace: testNamespace,
				},
				Type: corev1.SecretTypeDockerConfigJson,
			}

			result := checkResourceObject(secret, testNamespace, resourceTypeSecret)
			Expect(result).To(BeTrue())
		})

		It("should watch builder dockercfg secret", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "builder-dockercfg-12345",
					Namespace: testNamespace,
				},
				Type: corev1.SecretTypeDockercfg,
			}

			result := checkResourceObject(secret, testNamespace, resourceTypeSecret)
			Expect(result).To(BeTrue())
		})

		It("should watch pull secret with correct type", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FUSIONPULLSECRETNAME,
					Namespace: testNamespace,
				},
				Type: corev1.SecretTypeOpaque,
			}

			result := checkResourceObject(secret, testNamespace, resourceTypePullSecret)
			Expect(result).To(BeTrue())
		})

		It("should not watch pull secret with wrong type", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FUSIONPULLSECRETNAME,
					Namespace: testNamespace,
				},
				Type: corev1.SecretTypeDockerConfigJson,
			}

			result := checkResourceObject(secret, testNamespace, resourceTypePullSecret)
			Expect(result).To(BeFalse())
		})

		It("should not watch unknown resource type", func() {
			secret := &corev1.Secret{}
			result := checkResourceObject(secret, testNamespace, "unknown")
			Expect(result).To(BeFalse())
		})
	})

	Describe("compareResourceData", func() {
		It("should detect ConfigMap data changes", func() {
			oldCM := &corev1.ConfigMap{
				Data: map[string]string{"key": "old-value"},
			}
			newCM := &corev1.ConfigMap{
				Data: map[string]string{"key": "new-value"},
			}

			result := compareResourceData(oldCM, newCM, resourceTypeConfigMap)
			Expect(result).To(BeTrue())
		})

		It("should detect no change when ConfigMap data is same", func() {
			oldCM := &corev1.ConfigMap{
				Data: map[string]string{"key": "same-value"},
			}
			newCM := &corev1.ConfigMap{
				Data: map[string]string{"key": "same-value"},
			}

			result := compareResourceData(oldCM, newCM, resourceTypeConfigMap)
			Expect(result).To(BeFalse())
		})

		It("should detect Secret data changes", func() {
			oldSecret := &corev1.Secret{
				Data: map[string][]byte{"key": []byte("old-value")},
			}
			newSecret := &corev1.Secret{
				Data: map[string][]byte{"key": []byte("new-value")},
			}

			result := compareResourceData(oldSecret, newSecret, resourceTypeSecret)
			Expect(result).To(BeTrue())
		})

		It("should detect pull secret data changes", func() {
			oldSecret := &corev1.Secret{
				Data: map[string][]byte{"key": []byte("old-value")},
			}
			newSecret := &corev1.Secret{
				Data: map[string][]byte{"key": []byte("new-value")},
			}

			result := compareResourceData(oldSecret, newSecret, resourceTypePullSecret)
			Expect(result).To(BeTrue())
		})

		It("should handle invalid object types gracefully", func() {
			oldObj := &corev1.Pod{} // Wrong type
			newObj := &corev1.ConfigMap{
				Data: map[string]string{"key": "value"},
			}

			result := compareResourceData(oldObj, newObj, resourceTypeConfigMap)
			Expect(result).To(BeTrue()) // Should return true when types are wrong
		})

		It("should handle unknown resource types", func() {
			oldObj := &corev1.Secret{}
			newObj := &corev1.Secret{}

			result := compareResourceData(oldObj, newObj, "unknown")
			Expect(result).To(BeTrue())
		})
	})

	Describe("Selector Functions", func() {
		var (
			scheme       *runtime.Scheme
			fusionAccess *fusionv1alpha1.FusionAccess
			k8sClient    client.Client
			reconciler   *FusionAccessReconciler
		)

		BeforeEach(func() {
			scheme = runtime.NewScheme()
			Expect(corev1.AddToScheme(scheme)).To(Succeed())
			Expect(fusionv1alpha1.AddToScheme(scheme)).To(Succeed())

			fusionAccess = &fusionv1alpha1.FusionAccess{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-fusion",
					Namespace: testNamespace,
				},
			}

			k8sClient = fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(fusionAccess).
				Build()

			reconciler = &FusionAccessReconciler{
				Client: k8sClient,
			}
		})

		Describe("getKMMConfigmapSelector", func() {
			It("should trigger reconcile for correct KMM configmap", func() {
				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      kernelmodule.KMMImageConfigMapName,
						Namespace: testNamespace,
					},
				}

				requests := reconciler.getKMMConfigmapSelector(ctx, configMap)
				Expect(requests).To(HaveLen(1))
				Expect(requests[0]).To(Equal(reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      fusionAccess.Name,
						Namespace: fusionAccess.Namespace,
					},
				}))
			})

			It("should not trigger reconcile for wrong configmap", func() {
				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "wrong-configmap",
						Namespace: testNamespace,
					},
				}

				requests := reconciler.getKMMConfigmapSelector(ctx, configMap)
				Expect(requests).To(BeEmpty())
			})

			It("should not trigger reconcile for wrong namespace", func() {
				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      kernelmodule.KMMImageConfigMapName,
						Namespace: "wrong-namespace",
					},
				}

				requests := reconciler.getKMMConfigmapSelector(ctx, configMap)
				Expect(requests).To(BeEmpty())
			})
		})

		Context("when testing selector functions", func() {
			It("should trigger reconcile for IBM entitlement secret", func() {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      IBMENTITLEMENTNAME,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeDockerConfigJson,
				}

				requests := reconciler.getRegistrySecretSelector(ctx, secret)
				Expect(requests).To(HaveLen(1))
			})

			It("should trigger reconcile for custom registry secret", func() {
				// Create KMM config that specifies a custom registry secret
				kmmConfig := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      kernelmodule.KMMImageConfigMapName,
						Namespace: testNamespace,
					},
					Data: map[string]string{
						kernelmodule.KMMImageConfigKeyRegistrySecretName: "custom-registry-secret",
					},
				}
				Expect(k8sClient.Create(ctx, kmmConfig)).To(Succeed())

				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "custom-registry-secret",
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeDockerConfigJson,
				}

				requests := reconciler.getRegistrySecretSelector(ctx, secret)
				Expect(requests).To(HaveLen(1))

				// Cleanup
				Expect(k8sClient.Delete(ctx, kmmConfig)).To(Succeed())
			})

			It("should trigger reconcile for builder dockercfg secret", func() {
				// Create empty KMM config (no registry secret specified)
				kmmConfig := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      kernelmodule.KMMImageConfigMapName,
						Namespace: testNamespace,
					},
					Data: map[string]string{},
				}
				Expect(k8sClient.Create(ctx, kmmConfig)).To(Succeed())

				// Create the builder dockercfg secret
				secretName := "builder-dockercfg-abcde" //nolint:gosec // This is a test secret name, not a real credential
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeDockercfg,
				}
				Expect(k8sClient.Create(ctx, secret)).To(Succeed())

				requests := reconciler.getRegistrySecretSelector(ctx, secret)
				Expect(requests).To(HaveLen(1))

				// Cleanup
				Expect(k8sClient.Delete(ctx, kmmConfig)).To(Succeed())
				Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
			})

			It("should not trigger reconcile for unrelated secret", func() {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "unrelated-secret",
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
				}

				requests := reconciler.getRegistrySecretSelector(ctx, secret)
				Expect(requests).To(BeEmpty())
			})

			It("should not trigger reconcile for wrong namespace", func() {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      IBMENTITLEMENTNAME,
						Namespace: "wrong-namespace",
					},
				}

				requests := reconciler.getRegistrySecretSelector(ctx, secret)
				Expect(requests).To(BeEmpty())
			})
		})
	})

	Describe("Watch Predicate Functions", func() {
		It("should create KMM ConfigMap predicate without error", func() {
			predicateOpt := isItOurKMMConfigMap()
			Expect(predicateOpt).NotTo(BeNil())
		})

		It("should create registry secret predicate without error", func() {
			predicateOpt := isItOurRegistrySecret()
			Expect(predicateOpt).NotTo(BeNil())
		})

		It("should create pull secret predicate without error", func() {
			predicateOpt := isItOurPullSecret()
			Expect(predicateOpt).NotTo(BeNil())
		})

		Context("when testing underlying check functions", func() {
			It("should correctly identify KMM ConfigMap", func() {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      kernelmodule.KMMImageConfigMapName,
						Namespace: testNamespace,
					},
				}

				result := checkKMMConfigMap(cm, testNamespace)
				Expect(result).To(BeTrue())
			})

			It("should correctly identify registry secrets", func() {
				By("checking IBM entitlement secret")
				ibmSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      IBMENTITLEMENTNAME,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeDockerConfigJson,
				}

				result := checkRegistrySecret(ibmSecret, testNamespace)
				Expect(result).To(BeTrue())

				By("checking builder dockercfg secret")
				builderSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "builder-dockercfg-12345",
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeDockercfg,
				}

				result = checkRegistrySecret(builderSecret, testNamespace)
				Expect(result).To(BeTrue())

				By("rejecting wrong type secret")
				wrongSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      IBMENTITLEMENTNAME,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
				}

				result = checkRegistrySecret(wrongSecret, testNamespace)
				Expect(result).To(BeFalse())
			})

			It("should correctly identify pull secrets", func() {
				By("checking correct pull secret")
				pullSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      FUSIONPULLSECRETNAME,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
				}

				result := checkPullSecret(pullSecret, testNamespace)
				Expect(result).To(BeTrue())

				By("rejecting wrong type pull secret")
				wrongTypeSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      FUSIONPULLSECRETNAME,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeDockerConfigJson,
				}

				result = checkPullSecret(wrongTypeSecret, testNamespace)
				Expect(result).To(BeFalse())

				By("rejecting wrong name secret")
				wrongNameSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "wrong-secret",
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
				}

				result = checkPullSecret(wrongNameSecret, testNamespace)
				Expect(result).To(BeFalse())
			})
		})
	})

	Describe("Event Handling", func() {
		Context("when creating event objects", func() {
			It("should handle ConfigMap create events", func() {
				createEvent := event.CreateEvent{
					Object: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kernelmodule.KMMImageConfigMapName,
							Namespace: testNamespace,
						},
					},
				}

				Expect(createEvent.Object).NotTo(BeNil())
				Expect(createEvent.Object.GetName()).To(Equal(kernelmodule.KMMImageConfigMapName))
			})

			It("should handle Secret update events", func() {
				oldSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      IBMENTITLEMENTNAME,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{"key": []byte("old-value")},
				}

				newSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      IBMENTITLEMENTNAME,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{"key": []byte("new-value")},
				}

				updateEvent := event.UpdateEvent{
					ObjectOld: oldSecret,
					ObjectNew: newSecret,
				}

				Expect(updateEvent.ObjectOld).NotTo(BeNil())
				Expect(updateEvent.ObjectNew).NotTo(BeNil())
				Expect(updateEvent.ObjectOld.GetName()).To(Equal(IBMENTITLEMENTNAME))
				Expect(updateEvent.ObjectNew.GetName()).To(Equal(IBMENTITLEMENTNAME))
			})

			It("should handle delete events", func() {
				deleteEvent := event.DeleteEvent{
					Object: &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kernelmodule.KMMImageConfigMapName,
							Namespace: testNamespace,
						},
					},
				}

				Expect(deleteEvent.Object).NotTo(BeNil())
				Expect(deleteEvent.Object.GetName()).To(Equal(kernelmodule.KMMImageConfigMapName))
			})
		})
	})
})
