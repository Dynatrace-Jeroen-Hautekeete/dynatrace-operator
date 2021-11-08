package kubeobjects

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func AddToOwnerReference(objectMeta *metav1.ObjectMeta, existingReference metav1.OwnerReference) []metav1.OwnerReference {
	for _, ownerReference := range objectMeta.OwnerReferences {
		if ownerReference.UID == existingReference.UID {
			// Dynakube already defined as Owner of CSI DaemonSet
			return nil
		}
	}
	return append(objectMeta.OwnerReferences, existingReference)
}

func CreateOwnerReference(dynakube *dynatracev1beta1.DynaKube) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         dynakube.APIVersion,
		Kind:               dynakube.Kind,
		Name:               dynakube.Name,
		UID:                dynakube.UID,
		Controller:         pointer.BoolPtr(false),
		BlockOwnerDeletion: pointer.BoolPtr(false),
	}
}
