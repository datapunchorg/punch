package gcplib

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/common"
	"github.com/datapunchorg/punch/pkg/kubelib"
)

func GetServiceLoadBalancerHostPorts(kubeConfigFile string, projectId, zone, clusterName string, namespace string, serviceName string) ([]common.HostPort, error) {
	_, clientset, err := CreateKubernetesClient(kubeConfigFile, projectId, zone, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}
	hostPorts, err := kubelib.GetServiceLoadBalancerHostPorts(clientset, namespace, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get load balancers for service %s in namespace %s in cluster %s: %v", serviceName, namespace, clusterName, err)
	}
	return hostPorts, nil
}
