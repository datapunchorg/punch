/*
Copyright 2022 DataPunch Organization

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package superset

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	v1 "k8s.io/api/core/v1"
)

func DeploySupersetService(commandEnvironment framework.CommandEnvironment, supersetTopologySpec SupersetTopologySpec, region string, eksClusterName string) (string, error) {
	_, clientset, err := awslib.CreateKubernetesClient(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return "", fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	err = InstallSupersetHelm(commandEnvironment, supersetTopologySpec, region, eksClusterName)
	if err != nil {
		return "", fmt.Errorf("failed to install Superset helm chart: %s", err.Error())
	}

	namespace := supersetTopologySpec.Namespace
	podNamePrefix := "superset-worker"
	err = kubelib.WaitPodsInPhases(clientset, namespace, podNamePrefix, []v1.PodPhase{v1.PodRunning})
	if err != nil {
		return "", fmt.Errorf("pod %s*** in namespace %s is not in phase %s", podNamePrefix, namespace, v1.PodRunning)
	}

	serviceName := "superset"
	hostPorts, err := awslib.WaitServiceLoadBalancerHostPorts(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName, namespace, serviceName)
	if err != nil {
		return "", err
	}

	for _, entry := range hostPorts {
		// TODO do not hard code port 8088
		if entry.Port == 8088 {
			url := fmt.Sprintf("http://%s:%d", entry.Host, entry.Port)
			return url, nil
		}
	}

	return "", fmt.Errorf("did not find load balancer url for %s service in namespace %s", serviceName, namespace)
}

func InstallSupersetHelm(commandEnvironment framework.CommandEnvironment, supersetTopologySpec SupersetTopologySpec, region string, eksClusterName string) error {
	kubeConfig, err := awslib.CreateKubeConfig(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := supersetTopologySpec.HelmInstallName
	installNamespace := supersetTopologySpec.Namespace

	arguments := []string {
		"--set",
		"service.type=LoadBalancer",
	}

	err = kubelib.InstallHelm(commandEnvironment.Get(framework.CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvSupersetHelmChart), kubeConfig, arguments, installName, installNamespace)
	return err
}
