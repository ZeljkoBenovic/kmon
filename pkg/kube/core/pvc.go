package core

import (
	"context"
	"fmt"
	"log/slog"

	v3 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	v2 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned/typed/volumesnapshot/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type PVCManager interface {
	// Get fetches a PVC
	Get(namespace, name string) (*corev1.PersistentVolumeClaim, error)
	// Create creates a PVC
	Create(namespace, name string, opts ...PVCOptions) (*corev1.PersistentVolumeClaim, error)
	// Delete deletes a PVC
	Delete(namespace, name string) error
	// CreateVolumeSnapshotFromPVC crates a new PVC using the provided snapshot class name and snapshot name
	CreateVolumeSnapshotFromPVC(namespace string, name string, snapshotClassName string, sourcePVCName string) (*v3.VolumeSnapshot, error)
}

type pvc struct {
	ctx  context.Context
	log  *slog.Logger
	core v1.CoreV1Interface
	snap v2.VolumeSnapshotsGetter
}
type PVCOptions func(*corev1.PersistentVolumeClaim)

func WithStorageClassName(storageClassName string) PVCOptions {
	return func(pvc *corev1.PersistentVolumeClaim) {
		pvc.Spec.StorageClassName = &storageClassName
	}
}
func WithRestoreFromVolumeSnapshot(snapshotName string) PVCOptions {
	apiGr := "snapshot.storage.k8s.io"

	return func(pvc *corev1.PersistentVolumeClaim) {
		pvc.Spec.DataSource = &corev1.TypedLocalObjectReference{
			APIGroup: &apiGr,
			Kind:     "VolumeSnapshot",
			Name:     snapshotName,
		}
	}
}

func (p *pvc) Get(namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	p.log.Info("getting pvc", "namespace", namespace, "name", name)

	return p.core.PersistentVolumeClaims(namespace).Get(p.ctx, name, metav1.GetOptions{})
}

func (p *pvc) CreateFromSnapshot() {

}

func (p *pvc) Create(namespace, name string, opts ...PVCOptions) (*corev1.PersistentVolumeClaim, error) {
	p.log.Info("creating pvc", "namespace", namespace, "name", name)

	pvcObject := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("5Gi"),
				},
			},
		},
	}

	for _, opt := range opts {
		opt(pvcObject)
	}

	return p.core.PersistentVolumeClaims(namespace).Create(p.ctx, pvcObject, metav1.CreateOptions{})
}

func (p *pvc) Delete(namespace, name string) error {
	p.log.Info("deleting pvc", "namespace", namespace, "name", name)

	return p.core.PersistentVolumeClaims(namespace).Delete(p.ctx, name, metav1.DeleteOptions{})
}

func (p *pvc) CreateVolumeSnapshotFromPVC(namespace string, name string, snapshotClassName string, sourcePVCName string) (*v3.VolumeSnapshot, error) {
	p.log.Info("creating pvc", "namespace", namespace, "name", name)

	var snapClassName *string
	if snapshotClassName != "" {
		snapClassName = &snapshotClassName
	}

	vs, err := p.snap.VolumeSnapshots(namespace).Create(p.ctx, &v3.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", name),
			Namespace:    namespace,
		},
		Spec: v3.VolumeSnapshotSpec{
			Source: v3.VolumeSnapshotSource{
				PersistentVolumeClaimName: &sourcePVCName,
			},
			VolumeSnapshotClassName: snapClassName,
		},
		Status: nil,
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return vs, nil
}
