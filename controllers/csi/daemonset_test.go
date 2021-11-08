package dtcsi

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace       = "test-namespace"
	testDynakube        = "test-dynakube"
	testOperatorPodName = "test-operator-pod"
	testOperatorImage   = "test-operator-image"
)

func TestReconcile_NoOperatorPod(t *testing.T) {
	log := logger.NewDTLogger()
	fakeClient := fake.NewClient()
	rec := NewReconciler(fakeClient, scheme.Scheme, log, nil, testOperatorPodName, testNamespace)

	result, err := rec.Reconcile()
	require.Error(t, err)
	require.False(t, result)
}

func TestReconcile_NoOperatorImage(t *testing.T) {
	log := logger.NewDTLogger()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testOperatorPodName,
			Namespace: testNamespace,
		},
	}
	fakeClient := fake.NewClient(pod)
	rec := NewReconciler(fakeClient, scheme.Scheme, log, nil, testOperatorPodName, testNamespace)

	result, err := rec.Reconcile()
	require.Error(t, err)
	require.False(t, result)
}

func TestReconcile_CreateDaemonSet(t *testing.T) {
	log := logger.NewDTLogger()
	fakeClient := prepareFakeClient("", "")
	dk := prepareDynakube(testDynakube)
	rec := NewReconciler(fakeClient, scheme.Scheme, log, dk, testOperatorPodName, testNamespace)

	result, err := rec.Reconcile()
	require.NoError(t, err)
	assert.True(t, result)

	createdDaemonSet := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      DaemonSetName,
	}, createdDaemonSet)
	require.NoError(t, err)

	t.Run("metadata valid", func(t *testing.T) {
		assert.Len(t, createdDaemonSet.Labels, 1)

		assert.NotNil(t, createdDaemonSet.Annotations)
		assert.Contains(t, createdDaemonSet.Annotations, kubeobjects.AnnotationHash)

		assert.NotNil(t, createdDaemonSet.OwnerReferences)
		assert.Len(t, createdDaemonSet.OwnerReferences, 1)
	})
	t.Run("containers valid", func(t *testing.T) {
		assert.Len(t, createdDaemonSet.Spec.Template.Spec.Containers, 3)
	})
	t.Run("driver container valid", func(t *testing.T) {
		driver := createdDaemonSet.Spec.Template.Spec.Containers[0]
		assert.Equal(t, driver.Name, "driver")

		assert.Len(t, driver.Args, 4)

		assert.Len(t, driver.Ports, 1)

		assert.Len(t, driver.Env, 2)

		assert.NotNil(t, driver.Resources)
		assert.NotNil(t, driver.Resources.Requests)
		assert.Len(t, driver.Resources.Requests, 2)
		testQuantity(t, driver.Resources.Requests, corev1.ResourceCPU, "300m")
		testQuantity(t, driver.Resources.Requests, corev1.ResourceMemory, "100M")

		assert.NotNil(t, driver.Resources.Limits)
		assert.Len(t, driver.Resources.Limits, 2)
		testQuantity(t, driver.Resources.Limits, corev1.ResourceCPU, "300m")
		testQuantity(t, driver.Resources.Limits, corev1.ResourceMemory, "100M")

		assert.NotNil(t, driver.LivenessProbe)

		assert.NotNil(t, driver.SecurityContext)

		assert.Len(t, driver.VolumeMounts, 4)
	})
	t.Run("registrar container valid", func(t *testing.T) {
		registrar := createdDaemonSet.Spec.Template.Spec.Containers[1]
		assert.Equal(t, registrar.Name, "registrar")

		assert.Len(t, registrar.Args, 3)

		assert.Len(t, registrar.Ports, 1)

		assert.NotNil(t, registrar.Resources)
		assert.NotNil(t, registrar.Resources.Requests)
		assert.Len(t, registrar.Resources.Requests, 2)
		testQuantity(t, registrar.Resources.Requests, corev1.ResourceCPU, "10m")
		testQuantity(t, registrar.Resources.Requests, corev1.ResourceMemory, "15M")

		assert.NotNil(t, registrar.Resources.Limits)
		assert.Len(t, registrar.Resources.Limits, 2)
		testQuantity(t, registrar.Resources.Limits, corev1.ResourceCPU, "10m")
		testQuantity(t, registrar.Resources.Limits, corev1.ResourceMemory, "18M")

		assert.NotNil(t, registrar.LivenessProbe)

		assert.Len(t, registrar.VolumeMounts, 2)
	})
	t.Run("liveness probe container valid", func(t *testing.T) {
		livenessProbe := createdDaemonSet.Spec.Template.Spec.Containers[2]
		assert.Equal(t, livenessProbe.Name, "liveness-probe")

		assert.Len(t, livenessProbe.Args, 2)

		assert.NotNil(t, livenessProbe.Resources)
		assert.NotNil(t, livenessProbe.Resources.Requests)
		assert.Len(t, livenessProbe.Resources.Requests, 2)
		testQuantity(t, livenessProbe.Resources.Requests, corev1.ResourceCPU, "5m")
		testQuantity(t, livenessProbe.Resources.Requests, corev1.ResourceMemory, "15M")

		assert.NotNil(t, livenessProbe.Resources.Limits)
		assert.Len(t, livenessProbe.Resources.Limits, 2)
		testQuantity(t, livenessProbe.Resources.Limits, corev1.ResourceCPU, "5m")
		testQuantity(t, livenessProbe.Resources.Limits, corev1.ResourceMemory, "18M")

		assert.Len(t, livenessProbe.VolumeMounts, 1)
	})
	t.Run("volumes valid", func(t *testing.T) {
		assert.Len(t, createdDaemonSet.Spec.Template.Spec.Volumes, 5)
	})

	t.Run("tolerations valid", func(t *testing.T) {
		assert.Len(t, createdDaemonSet.Spec.Template.Spec.Tolerations, 0)
	})
}

func TestReconcile_CreateDaemonSet_TolerationFeatureFlag(t *testing.T) {
	tolerationAnnotation := "[{\"key\":\"node-role.kubernetes.io/master\",\"operator\":\"Exists\",\"effect\":\"NoSchedule\",\"tolerationSeconds\":420},{\"key\":\"test.test\",\"operator\":\"Equal\",\"effect\":\"NoSchedule\",\"value\":\"test\"}]"
	log := logger.NewDTLogger()
	fakeClient := prepareFakeClient("", tolerationAnnotation)
	dk := prepareDynakube(testDynakube)
	rec := NewReconciler(fakeClient, scheme.Scheme, log, dk, testOperatorPodName, testNamespace)

	result, err := rec.Reconcile()
	require.NoError(t, err)
	assert.True(t, result)

	createdDaemonSet := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      DaemonSetName,
	}, createdDaemonSet)
	require.NoError(t, err)

	t.Run("tolerations valid", func(t *testing.T) {
		assert.Len(t, createdDaemonSet.Spec.Template.Spec.Tolerations, 2)

		assert.Equal(t, "node-role.kubernetes.io/master", createdDaemonSet.Spec.Template.Spec.Tolerations[0].Key)
		assert.Equal(t, corev1.TolerationOpExists, createdDaemonSet.Spec.Template.Spec.Tolerations[0].Operator)
		assert.Equal(t, corev1.TaintEffectNoSchedule, createdDaemonSet.Spec.Template.Spec.Tolerations[0].Effect)
		assert.Equal(t, int64(420), *createdDaemonSet.Spec.Template.Spec.Tolerations[0].TolerationSeconds)

		assert.Equal(t, "test.test", createdDaemonSet.Spec.Template.Spec.Tolerations[1].Key)
		assert.Equal(t, corev1.TolerationOpEqual, createdDaemonSet.Spec.Template.Spec.Tolerations[1].Operator)
		assert.Equal(t, corev1.TaintEffectNoSchedule, createdDaemonSet.Spec.Template.Spec.Tolerations[1].Effect)
		assert.Equal(t, "test", createdDaemonSet.Spec.Template.Spec.Tolerations[1].Value)

	})
}

func testQuantity(t *testing.T, resourceList corev1.ResourceList, key corev1.ResourceName, quantity string) {
	assert.Contains(t, resourceList, key)
	expected, err := resource.ParseQuantity(quantity)
	require.NoError(t, err)
	assert.Equal(t, expected, resourceList[key])
}

func TestReconcile_UpdateDaemonSet(t *testing.T) {
	log := logger.NewDTLogger()
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DaemonSetName,
			Namespace: testNamespace,
			Annotations: map[string]string{
				kubeobjects.AnnotationHash: "old",
			},
		},
	}
	fakeClient := prepareFakeClient("", "", ds)

	dk := prepareDynakube(testDynakube)
	rec := NewReconciler(fakeClient, scheme.Scheme, log, dk, testOperatorPodName, testNamespace)
	result, err := rec.Reconcile()
	require.NoError(t, err)
	assert.True(t, result)

	updatedDaemonSet := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      DaemonSetName,
	}, updatedDaemonSet)
	require.NoError(t, err)

	assert.NotNil(t, updatedDaemonSet.Annotations)
	assert.Contains(t, updatedDaemonSet.Annotations, kubeobjects.AnnotationHash)
	assert.NotEqual(t, "old", updatedDaemonSet.Annotations[kubeobjects.AnnotationHash])
}

func TestReconcile_CSIResourceAnnotation(t *testing.T) {
	log := logger.NewDTLogger()
	fakeClient := prepareFakeClient("{\"driver\":{\"cpu\":\"99m\",\"memory\":\"99Mi\"},"+
		"\"registrar\":{\"cpu\":\"99m\",\"memory\":\"99Mi\"},"+
		"\"liveness-probe\":{\"cpu\":\"99m\",\"memory\":\"99Mi\"}}", "")
	dk := prepareDynakube(testDynakube)
	rec := NewReconciler(fakeClient, scheme.Scheme, log, dk, testOperatorPodName, testNamespace)

	result, err := rec.Reconcile()
	require.NoError(t, err)
	assert.True(t, result)

	createdDaemonSet := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      DaemonSetName,
	}, createdDaemonSet)
	require.NoError(t, err)

	driver := createdDaemonSet.Spec.Template.Spec.Containers[0]
	require.NotNil(t, driver.Resources.Requests)
	testQuantity(t, driver.Resources.Requests, corev1.ResourceCPU, "99m")
	testQuantity(t, driver.Resources.Requests, corev1.ResourceMemory, "99Mi")
	require.NotNil(t, driver.Resources.Limits)
	testQuantity(t, driver.Resources.Limits, corev1.ResourceCPU, "99m")
	testQuantity(t, driver.Resources.Limits, corev1.ResourceMemory, "99Mi")

	registrar := createdDaemonSet.Spec.Template.Spec.Containers[1]
	require.NotNil(t, registrar.Resources.Requests)
	testQuantity(t, registrar.Resources.Requests, corev1.ResourceCPU, "99m")
	testQuantity(t, registrar.Resources.Requests, corev1.ResourceMemory, "99Mi")
	require.NotNil(t, registrar.Resources.Limits)
	testQuantity(t, registrar.Resources.Limits, corev1.ResourceCPU, "99m")
	testQuantity(t, registrar.Resources.Limits, corev1.ResourceMemory, "99Mi")

	livenessProbe := createdDaemonSet.Spec.Template.Spec.Containers[2]
	require.NotNil(t, livenessProbe.Resources.Requests)
	testQuantity(t, livenessProbe.Resources.Requests, corev1.ResourceCPU, "99m")
	testQuantity(t, livenessProbe.Resources.Requests, corev1.ResourceMemory, "99Mi")
	require.NotNil(t, livenessProbe.Resources.Limits)
	testQuantity(t, livenessProbe.Resources.Limits, corev1.ResourceCPU, "99m")
	testQuantity(t, livenessProbe.Resources.Limits, corev1.ResourceMemory, "99Mi")
}

func prepareFakeClient(resourceAnnotation, tolerationsAnnotation string, objs ...client.Object) client.Client {
	trueVal := true

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "deployment",
			Namespace:   testNamespace,
			Annotations: map[string]string{},
		},
	}
	if resourceAnnotation != "" {
		deployment.Annotations[AnnotationCSIResourcesIdentifier] = resourceAnnotation
	}

	if tolerationsAnnotation != "" {
		deployment.Annotations[AnnotationCSITolerations] = tolerationsAnnotation
	}

	replicaSet := &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "ReplicaSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "replicaset",
			Namespace: testNamespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: deployment.APIVersion,
					Kind:       deployment.Kind,
					Name:       deployment.Name,
					UID:        deployment.UID,
					Controller: &trueVal,
				},
			},
		},
	}

	objs = append(objs,
		deployment,
		replicaSet,
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testOperatorPodName,
				Namespace: testNamespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: replicaSet.APIVersion,
						Kind:       replicaSet.Kind,
						Name:       replicaSet.Name,
						UID:        replicaSet.UID,
						Controller: &trueVal,
					},
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: testOperatorImage,
					},
				},
			},
		})
	return fake.NewClient(objs...)
}

func prepareDynakube(name string) *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DynaKube",
			APIVersion: "dynatrace.com/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
			UID:       types.UID(name),
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{},
		},
	}
}
