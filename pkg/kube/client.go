package kube

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	vol "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	v2 "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned/typed/volumesnapshot/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Client struct {
	v1.CoreV1Interface
	v2.VolumeSnapshotsGetter
}

func NewKubeClient(kubeContext string) (*Client, error) {
	var kubeConf *rest.Config

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not find user home dir: %w", err)
	}

	kubeconfigPath := filepath.Join(homeDir, ".kube", "config")

	if _, err = os.Stat(kubeconfigPath); errors.Is(err, os.ErrNotExist) {
		kubeConf, err = clientcmd.BuildConfigFromFlags("", "")
	} else {
		kconf, err := clientcmd.LoadFromFile(kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("could not load kubeconfig file: %w", err)
		}

		kubeConf, err = clientcmd.NewDefaultClientConfig(*kconf, &clientcmd.ConfigOverrides{
			CurrentContext: kubeContext,
		}).ClientConfig()

		kubeConf, err = clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	}
	if err != nil {
		return nil, fmt.Errorf("could not build kube config: %w", err)
	}

	kcl, err := kubernetes.NewForConfig(kubeConf)
	if err != nil {
		return nil, fmt.Errorf("could not build kube client: %w", err)
	}

	vcl, err := vol.NewForConfig(kubeConf)
	if err != nil {
		return nil, fmt.Errorf("could not build volumesnapshot client: %w", err)
	}

	return &Client{
		CoreV1Interface:       kcl.CoreV1(),
		VolumeSnapshotsGetter: vcl.SnapshotV1(),
	}, nil
}
