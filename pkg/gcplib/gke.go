package gcplib

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/datapunchorg/punch/pkg/common"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"google.golang.org/api/container/v1"
	"io/fs"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

func CreateKubeConfig(kubeConfigFile string, projectId, zone, clusterName string) (kubelib.KubeConfig, error) {
	var kubeConfig kubelib.KubeConfig
	if kubeConfigFile != "" {
		kubeConfig = kubelib.KubeConfig{
			ConfigFile: kubeConfigFile,
		}
	} else {
		var err error
		kubeConfig, err = CreateGkeKubeConfig(projectId, zone, clusterName)
		if err != nil {
			return kubelib.KubeConfig{}, err
		}
	}
	return kubeConfig, nil
}

func CreateGkeKubernetesClient(projectId, zone string, clusterName string) (kubelib.KubeConfig, *kubernetes.Clientset, error) {
	kubeConfig, err := CreateGkeKubeConfig(projectId, zone, clusterName)
	if err != nil {
		return kubeConfig, nil, err
	}

	config := rest.Config{
		Host:            kubeConfig.ApiServer,
		BearerToken:     kubeConfig.KubeToken,
		TLSClientConfig: rest.TLSClientConfig{CAData: kubeConfig.CA},
	}

	clientset, err := kubernetes.NewForConfig(&config)
	if err != nil {
		return kubelib.KubeConfig{}, nil, fmt.Errorf("failed to create Kubernetes client for cluster %s (%s): %s", clusterName, kubeConfig.ApiServer, err.Error())
	}
	return kubeConfig, clientset, nil
}

func CreateKubernetesClient(kubeConfigFile string, projectId, zone, clusterName string) (kubelib.KubeConfig, *kubernetes.Clientset, error) {
	var kubeConfig kubelib.KubeConfig
	var clientset *kubernetes.Clientset
	if kubeConfigFile != "" {
		kubeConfig = kubelib.KubeConfig{
			ConfigFile: kubeConfigFile,
		}
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			return kubelib.KubeConfig{}, nil, err
		}
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			return kubelib.KubeConfig{}, nil, err
		}
	} else {
		var err error
		kubeConfig, clientset, err = CreateGkeKubernetesClient(projectId, zone, clusterName)
		if err != nil {
			return kubelib.KubeConfig{}, nil, err
		}
	}
	return kubeConfig, clientset, nil
}

func CreateGkeKubeConfig(projectId, zone, clusterName string) (kubelib.KubeConfig, error) {
	ctx := context.Background()
	containerService, err := container.NewService(ctx)
	if err != nil {
		return kubelib.KubeConfig{}, fmt.Errorf("failed to new container service client: %w", err)
	}
	projectsZonesClustersGetCall := containerService.Projects.Zones.Clusters.Get(projectId, zone, clusterName)
	getClusterResult, err := projectsZonesClustersGetCall.Do()
	if err != nil {
		return kubelib.KubeConfig{}, fmt.Errorf("failed to run %s: %w", common.GetReflectTypeName(projectsZonesClustersGetCall), err)
	}

	ca, err := base64.StdEncoding.DecodeString(getClusterResult.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return kubelib.KubeConfig{}, fmt.Errorf("failed to decode ClusterCaCertificate: %w", err)
	}
	caFile, err := os.CreateTemp("", "ca.txt")
	if err != nil {
		caFile.Close()
		return kubelib.KubeConfig{}, err
	}
	caFile.Close()
	ioutil.WriteFile(caFile.Name(), []byte(ca), fs.ModePerm)

	kubeConfig := kubelib.KubeConfig{
		ApiServer: fmt.Sprintf("https://%s", getClusterResult.Endpoint),
		CAFile:    caFile.Name(),
		CA:        ca,
		KubeToken: getClusterResult.MasterAuth.ClientKey,
	}

	// clientcmd.NewNonInteractiveClientConfig().ClientConfig()

	return kubeConfig, nil
}
