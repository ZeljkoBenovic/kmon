package core

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/homedir"
)

type PodManager interface {
	// Create will create a pod in a specified name in a specified namespace.
	// PodOptions are not specified, a default nicolaka/netshoot contianer will be created
	Create(namespace string, name string, ops ...PodOptions) (*corev1.Pod, error)
	// Delete deletes a pod in specified namespace with a specified name
	Delete(namespace, name string) error
	// Exec runs a specified command within a pod in a specified namespace with a specified name
	// and outputs it onto stdout
	Exec(namespace string, name string, cmd []string) error
	// WaitReady waits for the pod the become ready before proceeding
	WaitReady(namespace string, name string, timeoutSeconds int) error
	// WaitDeleted waits for the pod to get deleted before proceeding
	WaitDeleted(namespace string, name string, timeoutSeconds int) error
}

type pod struct {
	ctx  context.Context
	log  *slog.Logger
	core v1.CoreV1Interface
}

type PodOptions func(*corev1.Pod)

func WithLabels(labels map[string]string) PodOptions {
	return func(pod *corev1.Pod) {
		pod.Labels = labels
	}
}
func WithAnnotations(annotations map[string]string) PodOptions {
	return func(pod *corev1.Pod) {
		pod.Annotations = annotations
	}
}

func WithPVC(volumeName, mountPath, pvcName string) PodOptions {
	return func(pod *corev1.Pod) {
		pod.Spec.Volumes = []corev1.Volume{
			{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			},
		}

		pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{
				Name:      volumeName,
				MountPath: mountPath,
			},
		}
	}
}

func (p *pod) Create(namespace string, name string, opts ...PodOptions) (*corev1.Pod, error) {
	p.log.Info("creating pod", "namespace", namespace, "name", name)

	podDefinition := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "netshoot",
					Image:   "ghcr.io/nicolaka/netshoot:v0.14",
					Command: []string{"tail", "-f", "/dev/null"},
				},
			},
		},
	}

	for _, opt := range opts {
		opt(podDefinition)
	}

	return p.core.Pods(namespace).Create(p.ctx, podDefinition, metav1.CreateOptions{})
}

func (p *pod) WaitReady(namespace string, name string, timeoutSec int) error {
	p.log.Info("waiting for pod to become ready", "namespace", namespace, "name", name)

	podSelector := fields.SelectorFromSet(fields.Set{
		"metadata.name":      name,
		"metadata.namespace": namespace,
	})

	watcher, err := p.core.Pods(namespace).Watch(p.ctx, metav1.ListOptions{
		FieldSelector: podSelector.String(),
	})
	if err != nil {
		return fmt.Errorf("could not watch pod: %w", err)
	}

	for {
		select {
		case event := <-watcher.ResultChan():
			if event.Type == watch.Added || event.Type == watch.Modified {
				if event.Object.(*corev1.Pod).Status.Phase == corev1.PodRunning {
					return nil
				}
			}
		case <-time.After(time.Second * time.Duration(timeoutSec)):
			return fmt.Errorf("timeout waiting for pod to become running")
		}
	}
}

func (p *pod) WaitDeleted(namespace string, name string, timeoutSec int) error {
	p.log.Info("waiting for pod to be deleted", "namespace", namespace, "name", name)

	podSelector := fields.SelectorFromSet(fields.Set{
		"metadata.name":      name,
		"metadata.namespace": namespace,
	})

	watcher, err := p.core.Pods(namespace).Watch(p.ctx, metav1.ListOptions{
		FieldSelector: podSelector.String(),
	})
	if err != nil {
		return fmt.Errorf("could not watch pod: %w", err)
	}

	for {
		select {
		case event := <-watcher.ResultChan():
			if event.Type == watch.Deleted {
				return nil
			}
		case <-time.After(time.Second * time.Duration(timeoutSec)):
			return fmt.Errorf("timeout waiting for pod to be deleted")
		}
	}
}

func (p *pod) Delete(namespace, name string) error {
	p.log.Info("deleting pod", "namespace", namespace, "name", name)
	return p.core.Pods(namespace).Delete(p.ctx, name, metav1.DeleteOptions{})
}

func (p *pod) Exec(namespace string, name string, cmd []string) error {
	p.log.Info("executing pod", "namespace", namespace, "name", name)

	req := p.core.RESTClient().Post().Resource("pods").
		Name(name).Namespace(namespace).SubResource("exec")

	option := &corev1.PodExecOptions{
		Command: cmd,
		Stdin:   true,
		Stdout:  true,
		Stderr:  true,
		TTY:     true,
	}

	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)

	// TODO: should work from within the cluster
	client, err := clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	if err != nil {
		return fmt.Errorf("could not build kube client config: %w", err)
	}

	exec, err := remotecommand.NewSPDYExecutor(client, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("spdy executor failed: %w", err)
	}
	err = exec.StreamWithContext(p.ctx, remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
		return fmt.Errorf("streaming failed: %w", err)
	}

	return nil
}
