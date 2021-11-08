package dtcsi

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/controllers"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ConfigureCSIDriver(
	client client.Client, apiReader client.Reader, scheme *runtime.Scheme, operatorPodName, operatorNamespace string,
	dkState *controllers.DynakubeState, updateInterval time.Duration) error {

	if dkState.Instance.NeedsCSIDriver() {
		err := addDynakubeOwnerReference(client, apiReader, scheme, operatorPodName, operatorNamespace, dkState, updateInterval)
		if err != nil {
			return err
		}
	} else {
		err := removeDynakubeOwnerReference(client, dkState)
		if err != nil {
			return err
		}
	}
	return nil
}

// addDynakubeOwnerReference enables csi driver, by creating its DaemonSet (if it does not exist yet)
// and adds the current Dynakube to the OwnerReferences of the DaemonSet
func addDynakubeOwnerReference(
	client client.Client, apiReader client.Reader, scheme *runtime.Scheme, operatorPodName string, operatorNamespace string,
	dkState *controllers.DynakubeState, updateInterval time.Duration) error {

	_, err := createOrUpdateCSIDaemonSet(client, scheme, operatorPodName, operatorNamespace, dkState, updateInterval)
	if err != nil {
		return err
	}

	return nil
}

func createOrUpdateCSIDaemonSet(
	client client.Client, scheme *runtime.Scheme, operatorPodName string, operatorNamespace string,
	dkState *controllers.DynakubeState, updateInterval time.Duration) (bool, error) {

	dkState.Log.Info("enabling csi driver")
	csiDaemonSetReconciler := NewReconciler(client, scheme, dkState.Log, dkState.Instance, operatorPodName, operatorNamespace)
	upd, err := csiDaemonSetReconciler.Reconcile()
	if err != nil {
		return false, err
	}
	if dkState.Update(upd, updateInterval, "CSI driver reconciled") {
		return true, nil
	}
	return false, nil
}

// removeDynakubeOwnerReference removes the current Dynakube from the OwnerReferences of the DaemonSet
// and deletes the DaemonSet if no Owners are left.
func removeDynakubeOwnerReference(clt client.Client, dkState *controllers.DynakubeState) error {
	csiDaemonSet, err := getCSIDaemonSet(clt, dkState.Instance.Namespace)
	if k8serrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	itemIndex, itemFound := findOwnerReferenceIndex(csiDaemonSet.OwnerReferences, dkState.Instance.UID)
	if !itemFound {
		// Dynakube was not found in existing OwnerReferences
		return nil
	}

	err = updateOrDeleteCSIDaemonSet(clt, csiDaemonSet, itemIndex, dkState.Log)
	if err != nil {
		return err
	}
	return nil
}

func updateOrDeleteCSIDaemonSet(clt client.Client, csiDaemonSet *appsv1.DaemonSet, itemIndex int, log logr.Logger) error {
	if len(csiDaemonSet.OwnerReferences) > 1 {
		csiDaemonSet.OwnerReferences = append(
			csiDaemonSet.OwnerReferences[:itemIndex],
			csiDaemonSet.OwnerReferences[itemIndex+1:]...)
	} else {
		// Delete CSI DaemonSet manually if no OwnerReferences are left
		return clt.Delete(context.TODO(), csiDaemonSet)
	}

	log.Info("Removing Dynakube from CSI DaemonSet")
	return clt.Update(context.TODO(), csiDaemonSet)
}

func findOwnerReferenceIndex(ownerReferences []metav1.OwnerReference, instanceUID types.UID) (int, bool) {
	for i, ownerReference := range ownerReferences {
		if ownerReference.UID == instanceUID {
			return i, true
		}
	}
	return 0, false
}

func getCSIDaemonSet(apiReader client.Reader, namespace string) (*appsv1.DaemonSet, error) {
	csiDaemonSet := &appsv1.DaemonSet{}
	err := apiReader.Get(context.TODO(),
		client.ObjectKey{
			Name:      DaemonSetName,
			Namespace: namespace,
		}, csiDaemonSet)

	if err != nil {
		return nil, err
	}
	return csiDaemonSet, nil
}
