package core

import (
	"context"
	"log/slog"

	v2 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned/typed/volumesnapshot/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type KubeCore interface {
	v1.CoreV1Interface
	v2.VolumeSnapshotsGetter
}
type Core struct {
	pod *pod
	pvc *pvc
}

func NewCore(log *slog.Logger, ctx context.Context, cl KubeCore) *Core {
	var c Core

	c.pod = &pod{
		ctx:  ctx,
		log:  log.WithGroup("pod"),
		core: cl,
	}

	c.pvc = &pvc{
		ctx:  ctx,
		log:  log.WithGroup("pvc"),
		core: cl,
		snap: cl,
	}

	return &c
}

func (c *Core) Pod() PodManager {
	return c.pod
}

func (c *Core) PVC() PVCManager {
	return c.pvc
}
