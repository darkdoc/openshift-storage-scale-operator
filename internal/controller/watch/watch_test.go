package watch

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/openshift-storage-scale/openshift-fusion-access-operator/internal/controller/kernelmodule"
	"github.com/openshift-storage-scale/openshift-fusion-access-operator/internal/utils"
)

var _ = Describe("Watch Functionality", func() {
	const testNamespace = "test-namespace"

	BeforeEach(func() {
		os.Setenv("DEPLOYMENT_NAMESPACE", testNamespace)
	})

	AfterEach(func() {
		os.Unsetenv("DEPLOYMENT_NAMESPACE")
	})

	Describe("CheckResourceObject", func() {
		It("should watch KMM ConfigMap in correct namespace", func() {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kernelmodule.KMMImageConfigMapName,
					Namespace: testNamespace,
				},
			}

			result := CheckResourceObject(cm, testNamespace, utils.ResourceTypeConfigMap)
			Expect(result).To(BeTrue())
		})

		It("should not watch ConfigMap with wrong name", func() {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other-configmap",
					Namespace: testNamespace,
				},
			}

			result := CheckResourceObject(cm, testNamespace, utils.ResourceTypeConfigMap)
			Expect(result).To(BeFalse())
		})

		It("should not watch ConfigMap in wrong namespace", func() {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kernelmodule.KMMImageConfigMapName,
					Namespace: "wrong-namespace",
				},
			}

			result := CheckResourceObject(cm, testNamespace, utils.ResourceTypeConfigMap)
			Expect(result).To(BeFalse())
		})

		It("should watch IBM entitlement secret", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.IBMEntitlementSecretName,
					Namespace: testNamespace,
				},
				Type: corev1.SecretTypeDockerConfigJson,
			}

			result := CheckResourceObject(secret, testNamespace, utils.ResourceTypeSecret)
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

			result := CheckResourceObject(secret, testNamespace, utils.ResourceTypeSecret)
			Expect(result).To(BeTrue())
		})

		It("should watch pull secret with correct type", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.FusionPullSecretName,
					Namespace: testNamespace,
				},
				Type: corev1.SecretTypeOpaque,
			}

			result := CheckResourceObject(secret, testNamespace, utils.ResourceTypePullSecret)
			Expect(result).To(BeTrue())
		})

		It("should not watch pull secret with wrong type", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      utils.FusionPullSecretName,
					Namespace: testNamespace,
				},
				Type: corev1.SecretTypeDockerConfigJson,
			}

			result := CheckResourceObject(secret, testNamespace, utils.ResourceTypePullSecret)
			Expect(result).To(BeFalse())
		})

		It("should not watch unknown resource type", func() {
			secret := &corev1.Secret{}
			result := CheckResourceObject(secret, testNamespace, "unknown")
			Expect(result).To(BeFalse())
		})
	})

	Describe("CompareResourceData", func() {
		It("should detect ConfigMap data changes", func() {
			oldCM := &corev1.ConfigMap{
				Data: map[string]string{"key": "old-value"},
			}
			newCM := &corev1.ConfigMap{
				Data: map[string]string{"key": "new-value"},
			}

			result := CompareResourceData(oldCM, newCM, utils.ResourceTypeConfigMap)
			Expect(result).To(BeTrue())
		})

		It("should detect no change when ConfigMap data is same", func() {
			oldCM := &corev1.ConfigMap{
				Data: map[string]string{"key": "same-value"},
			}
			newCM := &corev1.ConfigMap{
				Data: map[string]string{"key": "same-value"},
			}

			result := CompareResourceData(oldCM, newCM, utils.ResourceTypeConfigMap)
			Expect(result).To(BeFalse())
		})

		It("should detect Secret data changes", func() {
			oldSecret := &corev1.Secret{
				Data: map[string][]byte{"key": []byte("old-value")},
			}
			newSecret := &corev1.Secret{
				Data: map[string][]byte{"key": []byte("new-value")},
			}

			result := CompareResourceData(oldSecret, newSecret, utils.ResourceTypeSecret)
			Expect(result).To(BeTrue())
		})

		It("should detect pull secret data changes", func() {
			oldSecret := &corev1.Secret{
				Data: map[string][]byte{"key": []byte("old-value")},
			}
			newSecret := &corev1.Secret{
				Data: map[string][]byte{"key": []byte("new-value")},
			}

			result := CompareResourceData(oldSecret, newSecret, utils.ResourceTypePullSecret)
			Expect(result).To(BeTrue())
		})

		It("should handle invalid object types gracefully", func() {
			oldObj := &corev1.Pod{} // Wrong type
			newObj := &corev1.ConfigMap{
				Data: map[string]string{"key": "value"},
			}

			result := CompareResourceData(oldObj, newObj, utils.ResourceTypeConfigMap)
			Expect(result).To(BeTrue()) // Should return true when types are wrong
		})

		It("should handle unknown resource types", func() {
			oldObj := &corev1.Secret{}
			newObj := &corev1.Secret{}

			result := CompareResourceData(oldObj, newObj, "unknown")
			Expect(result).To(BeTrue())
		})
	})

	Describe("Watch Predicate Functions", func() {
		It("should create KMM ConfigMap predicate without error", func() {
			predicateOpt := IsItOurKMMConfigMap()
			Expect(predicateOpt).NotTo(BeNil())
		})

		It("should create registry secret predicate without error", func() {
			predicateOpt := IsItOurRegistrySecret()
			Expect(predicateOpt).NotTo(BeNil())
		})

		It("should create pull secret predicate without error", func() {
			predicateOpt := IsItOurPullSecret()
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

				result := CheckKMMConfigMap(cm, testNamespace)
				Expect(result).To(BeTrue())
			})

			It("should correctly identify registry secrets", func() {
				By("checking IBM entitlement secret")
				ibmSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      utils.IBMEntitlementSecretName,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeDockerConfigJson,
				}

				result := CheckRegistrySecret(ibmSecret, testNamespace)
				Expect(result).To(BeTrue())

				By("checking builder dockercfg secret")
				builderSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "builder-dockercfg-12345",
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeDockercfg,
				}

				result = CheckRegistrySecret(builderSecret, testNamespace)
				Expect(result).To(BeTrue())

				By("rejecting wrong type secret")
				wrongSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      utils.IBMEntitlementSecretName,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
				}

				result = CheckRegistrySecret(wrongSecret, testNamespace)
				Expect(result).To(BeFalse())
			})

			It("should correctly identify pull secrets", func() {
				By("checking correct pull secret")
				pullSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      utils.FusionPullSecretName,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
				}

				result := CheckPullSecret(pullSecret, testNamespace)
				Expect(result).To(BeTrue())

				By("rejecting wrong type pull secret")
				wrongTypeSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      utils.FusionPullSecretName,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeDockerConfigJson,
				}

				result = CheckPullSecret(wrongTypeSecret, testNamespace)
				Expect(result).To(BeFalse())

				By("rejecting wrong name secret")
				wrongNameSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "wrong-secret",
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeOpaque,
				}

				result = CheckPullSecret(wrongNameSecret, testNamespace)
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
						Name:      utils.IBMEntitlementSecretName,
						Namespace: testNamespace,
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{"key": []byte("old-value")},
				}

				newSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      utils.IBMEntitlementSecretName,
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
				Expect(updateEvent.ObjectOld.GetName()).To(Equal(utils.IBMEntitlementSecretName))
				Expect(updateEvent.ObjectNew.GetName()).To(Equal(utils.IBMEntitlementSecretName))
			})
		})
	})
})
