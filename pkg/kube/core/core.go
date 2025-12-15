package core

import (
	"context"
	"github.com/zeljkobenovic/kmon/pkg/kube"
	"log/slog"
)

type Core struct {
	pod *pod
	pvc *pvc
}

func NewCore(log *slog.Logger, ctx context.Context, cl *kube.Client) *Core {
	var c Core

	c.pod = &pod{
		ctx:  ctx,
		log:  log.WithGroup("pod"),
		core: cl,
	}

	c.pvc = &pvc{
		ctx:       ctx,
		log:       log.WithGroup("pvc"),
		core:      cl,
		snap:      cl,
		snapClass: cl,
	}

	return &c
}

func (c *Core) Pod() PodManager {
	return c.pod
}

func (c *Core) PVC() PVCManager {
	return c.pvc
}

func (c *Core) Storage() StorageManager {
	return c.pvc
}
