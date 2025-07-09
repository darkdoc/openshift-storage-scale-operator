package watch

import (
	"reflect"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/openshift-storage-scale/openshift-fusion-access-operator/internal/controller/kernelmodule"
	"github.com/openshift-storage-scale/openshift-fusion-access-operator/internal/utils"
)

// Watch functionality for FusionAccess controller"

// CheckResourceObject checks if a resource object should be watched based on its type
func CheckResourceObject(obj client.Object, ns, resourceType string) bool {
	switch resourceType {
	case utils.ResourceTypeConfigMap:
		cm, ok := obj.(*corev1.ConfigMap)
		if !ok {
			return false
		}
		return CheckKMMConfigMap(cm, ns)
	case utils.ResourceTypeSecret:
		secret, ok := obj.(*corev1.Secret)
		if !ok {
			return false
		}
		return CheckRegistrySecret(secret, ns)
	case utils.ResourceTypePullSecret:
		secret, ok := obj.(*corev1.Secret)
		if !ok {
			return false
		}
		return CheckPullSecret(secret, ns)
	default:
		return false
	}
}

// CompareResourceData compares the data of two resource objects
func CompareResourceData(oldObj, newObj client.Object, resourceType string) bool {
	switch resourceType {
	case utils.ResourceTypeConfigMap:
		oldCM, okOld := oldObj.(*corev1.ConfigMap)
		newCM, okNew := newObj.(*corev1.ConfigMap)
		if !okOld || !okNew {
			return true
		}
		return !reflect.DeepEqual(oldCM.Data, newCM.Data)
	case utils.ResourceTypeSecret, utils.ResourceTypePullSecret:
		oldSecret, okOld := oldObj.(*corev1.Secret)
		newSecret, okNew := newObj.(*corev1.Secret)
		if !okOld || !okNew {
			return true
		}
		return !reflect.DeepEqual(oldSecret.Data, newSecret.Data)
	default:
		return true
	}
}

// CreateResourcePredicate creates a generic predicate for watching resources
func CreateResourcePredicate(resourceType string) builder.WatchesOption {
	return builder.WithPredicates(predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			ns, err := utils.GetDeploymentNamespace()
			if err != nil {
				return false
			}
			return CheckResourceObject(e.Object, ns, resourceType)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			ns, err := utils.GetDeploymentNamespace()
			if err != nil {
				return false
			}
			if !CheckResourceObject(e.ObjectNew, ns, resourceType) {
				return false
			}
			return CompareResourceData(e.ObjectOld, e.ObjectNew, resourceType)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Pull secrets don't care about delete events
			if resourceType == utils.ResourceTypePullSecret {
				return false
			}
			ns, err := utils.GetDeploymentNamespace()
			if err != nil {
				return false
			}
			return CheckResourceObject(e.Object, ns, resourceType)
		},
		GenericFunc: func(_ event.GenericEvent) bool {
			return false
		},
	})
}

// CheckKMMConfigMap checks if a ConfigMap is the KMM config map we care about
func CheckKMMConfigMap(cm *corev1.ConfigMap, ns string) bool {
	if cm.Name != kernelmodule.KMMImageConfigMapName {
		return false
	}
	if cm.Namespace != ns {
		return false
	}
	return true
}

// CheckRegistrySecret checks if a Secret is a registry-related secret we care about
func CheckRegistrySecret(secret *corev1.Secret, ns string) bool {
	if secret.Namespace != ns {
		return false
	}

	// Check if it's IBM entitlement secret with correct type
	if secret.Name == utils.IBMEntitlementSecretName {
		return secret.Type == corev1.SecretTypeDockerConfigJson
	}

	// Check if it's a builder dockercfg secret
	builderSecretPattern := `^builder-dockercfg-.*$` //nolint:gosec // This is a regex pattern, not a credential
	matched, _ := regexp.MatchString(builderSecretPattern, secret.Name)
	if matched {
		return secret.Type == corev1.SecretTypeDockercfg
	}

	// Note: We can't easily check for the registry secret from config here without
	// making a client call, so we'll be conservative and watch more secrets than necessary.
	// The selector function will filter them properly.
	return secret.Type == corev1.SecretTypeDockerConfigJson || secret.Type == corev1.SecretTypeDockercfg
}

// CheckPullSecret checks if a Secret is the fusion pull secret we care about
func CheckPullSecret(secret *corev1.Secret, ns string) bool {
	if secret.Type != "Opaque" {
		return false
	}
	if secret.Name != utils.FusionPullSecretName {
		return false
	}
	if secret.Namespace != ns {
		return false
	}
	return true
}

// IsItOurPullSecret returns true for Create or changed Update events
func IsItOurPullSecret() builder.WatchesOption {
	return CreateResourcePredicate(utils.ResourceTypePullSecret)
}

// IsItOurKMMConfigMap returns true for Create or changed Update events on the KMM config map
func IsItOurKMMConfigMap() builder.WatchesOption {
	return CreateResourcePredicate(utils.ResourceTypeConfigMap)
}

// IsItOurRegistrySecret returns true for Create or changed Update events on registry-related secrets
func IsItOurRegistrySecret() builder.WatchesOption {
	return CreateResourcePredicate(utils.ResourceTypeSecret)
}
