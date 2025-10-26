package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/zeljkobenovic/kmon/pkg/config"
	"github.com/zeljkobenovic/kmon/pkg/kube"
	"github.com/zeljkobenovic/kmon/pkg/kube/core"
)

type App struct {
	core *core.Core
	conf *config.Config
	log  *slog.Logger
	ctx  context.Context
}

func NewApp() (*App, error) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	ctx := context.Background()

	c, err := config.NewConfig(log)
	if err != nil {
		return nil, err
	}

	kcl, err := kube.NewKubeClient(c.Namespace)
	if err != nil {
		return nil, err
	}

	return &App{
		core: core.NewCore(log, ctx, kcl),
		conf: c,
		log:  log.WithGroup("app"),
		ctx:  ctx,
	}, nil
}

func (a *App) Run() error {
	return a.conf.Execute(a)

}

func (a *App) PodCmdHandler() error {
	switch a.conf.Pod.Mode {
	case config.RunFromPVC:
		return a.runPodFromPVC()
	case config.RunFromSnapshot:
		return a.runPodFromSnapshot()
	default:
		return fmt.Errorf("invalid pod mode: %s", a.conf.Pod.Mode)
	}
}

func (a *App) PVCCmdHandler() error {
	switch a.conf.PVC.Mode {
	case config.SnapshotFromPVC:
		return a.createSnapshotFromPVC()
	case config.PVCfromSnapshot:
		return a.createPVCfromSnapshot()
	default:
		return fmt.Errorf("invalid pod mode: %s", a.conf.PVC.Mode)
	}
}

func (a *App) createTestPVC() error {
	pvc, err := a.core.PVC().Create(a.conf.Namespace, a.conf.PVC.Name)
	if err != nil {
		return fmt.Errorf("failed to create pvc: %s", err)
	}

	a.log.Info("pvc created", "name", pvc.Name, "time", pvc.CreationTimestamp.String())
	return nil
}

func (a *App) runPodFromPVC() error {
	pod, err := a.core.Pod().Create(
		a.conf.Namespace,
		a.conf.Pod.Name,
		core.WithPVC(
			a.conf.Pod.VolumeName,
			a.conf.Pod.MountPath,
			a.conf.Pod.PVCName,
		),
	)
	if err != nil {
		return fmt.Errorf("pod create failed: %v", err)
	}

	if err = a.core.Pod().WaitReady(pod.Namespace, pod.Name, 60); err != nil {
		return fmt.Errorf("pod wait ready failed: %v", err)
	}

	a.log.Info("pod successfully created", "name", pod.Name, "time", pod.CreationTimestamp.String())

	return nil
}

func (a *App) runPodFromSnapshot() error {
	pvc, err := a.core.PVC().Create(
		a.conf.Namespace,
		a.conf.PVC.Name,
		core.WithRestoreFromVolumeSnapshot(a.conf.Pod.SnapshotName),
	)
	if err != nil {
		return fmt.Errorf("failed to create pvc: %s", err)
	}

	a.log.Info("pvc created", "name", pvc.Name, "time", pvc.CreationTimestamp.String())

	pod, err := a.core.Pod().Create(
		a.conf.Namespace,
		a.conf.Pod.Name,
		core.WithPVC(
			a.conf.Pod.VolumeName,
			a.conf.Pod.MountPath,
			a.conf.Pod.PVCName,
		),
	)
	if err != nil {
		return fmt.Errorf("pod create failed: %v", err)
	}

	a.log.Info("pod created", "name", pod.Name, "time", pod.CreationTimestamp.String())

	return nil
}

func (a *App) runTest() error {
	pvc, err := a.core.PVC().Create(a.conf.Namespace, a.conf.PVC.Name)
	if err != nil {
		return err
	}

	a.log.Info("pvc created", "name", pvc.Name)

	pod, err := a.core.Pod().Create(
		a.conf.Namespace,
		a.conf.Pod.Name,
		core.WithPVC(
			a.conf.Pod.VolumeName,
			a.conf.Pod.MountPath,
			pvc.Name,
		),
	)
	if err != nil {
		return err
	}

	a.log.Info("pod created", "time", pod.CreationTimestamp.String())

	if err = a.core.Pod().WaitReady(pod.Namespace, pod.Name, 60); err != nil {
		return err
	}

	if err = a.core.Pod().Exec(pod.Namespace, pod.Name, []string{"ls", "-lah", "/"}); err != nil {
		return err
	}

	err = a.core.Pod().Delete(pod.Namespace, pod.Name)
	if err != nil {
		return err
	}

	err = a.core.Pod().WaitDeleted(pod.Namespace, pod.Name, 60)
	if err != nil {
		return err
	}

	a.log.Info("pod deleted", "time", pod.CreationTimestamp.String())

	if err = a.core.PVC().Delete(pvc.Namespace, pvc.Name); err != nil {
		return err
	}

	a.log.Info("pvc deleted", "name", pvc.Name)

	return nil
}

func (a *App) createSnapshotFromPVC() error {
	pvc, err := a.core.PVC().CreateVolumeSnapshotFromPVC(
		a.conf.Namespace,
		a.conf.PVC.SnapshotName,
		a.conf.PVC.SnapshotClassName,
		a.conf.PVC.SourcePVCName,
	)
	if err != nil {
		return fmt.Errorf("failed to create pvc snapshot: %s", err)
	}

	a.log.Info("pvc snapshot", "name", pvc.Name, "time", pvc.CreationTimestamp.String())

	return nil
}

func (a *App) createPVCfromSnapshot() error {
	pvc, err := a.core.PVC().Create(
		a.conf.Namespace,
		a.conf.PVC.Name,
		core.WithRestoreFromVolumeSnapshot(a.conf.Pod.SnapshotName),
	)
	if err != nil {
		return fmt.Errorf("failed to create pvc: %s", err)
	}

	a.log.Info("pvc created", "name", pvc.Name, "time", pvc.CreationTimestamp.String())

	return nil
}
