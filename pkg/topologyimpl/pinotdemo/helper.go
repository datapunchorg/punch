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

package pinotdemo

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"github.com/datapunchorg/punch/pkg/topologyimpl/sparkoneks"
	v1 "k8s.io/api/core/v1"
)

func DeployKyuubiServer(commandEnvironment framework.CommandEnvironment, kyuubiComponentSpec KyuubiComponentSpec, region string, eksClusterName string, sparkApiGateway sparkoneks.SparkApiGateway, sparkApiGatewayUrl string) (string, error) {
	_, clientset, err := awslib.CreateKubernetesClient(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return "", fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	err = InstallKyuubiHelm(commandEnvironment, kyuubiComponentSpec, region, eksClusterName, sparkApiGatewayUrl, sparkApiGateway.User, sparkApiGateway.Password)
	if err != nil {
		return "", fmt.Errorf("failed to install Spark History Server helm chart: %s", err.Error())
	}

	helmInstallName := kyuubiComponentSpec.HelmInstallName
	namespace := kyuubiComponentSpec.Namespace
	podNamePrefix := helmInstallName
	err = kubelib.WaitPodsInPhases(clientset, namespace, podNamePrefix, []v1.PodPhase{v1.PodRunning})
	if err != nil {
		return "", fmt.Errorf("pod %s*** in namespace %s is not in phase %s", podNamePrefix, namespace, v1.PodRunning)
	}

	serviceName := "kyuubi-svc"
	hostPorts, err := awslib.WaitServiceLoadBalancerHostPorts(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName, namespace, serviceName)
	if err != nil {
		return "", err
	}

	for _, entry := range hostPorts {
		// TODO do not hard code port 10009
		if entry.Port == 10009 {
			url := fmt.Sprintf("thrift://%s:%d", entry.Host, entry.Port)
			return url, nil
		}
	}

	return "", fmt.Errorf("did not find load balancer url for kyuubi thrift server")
}

func InstallKyuubiHelm(commandEnvironment framework.CommandEnvironment, kyuubiComponentSpec KyuubiComponentSpec, region string, eksClusterName string, sparkApiGatewayUrl string, apiGatewayUser string, apiGatewayPassword string) error {
	kubeConfig, err := awslib.CreateKubeConfig(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := kyuubiComponentSpec.HelmInstallName
	installNamespace := kyuubiComponentSpec.Namespace

	serviceRegistryRestUrl := fmt.Sprintf(
		"http://kyuubi-server-0.kyuubi-svc.%s.svc.cluster.local:10099/api/v1",
		installNamespace)

	arguments := []string{
		"--set", fmt.Sprintf("image.repository=%s", kyuubiComponentSpec.ImageRepository),
		"--set", fmt.Sprintf("image.tag=%s", kyuubiComponentSpec.ImageTag),
		"--set", fmt.Sprintf("serviceRegistry.restUrl=%s", serviceRegistryRestUrl),
		"--set", fmt.Sprintf("sparkSqlEngine.jarFile=%s", kyuubiComponentSpec.SparkSqlEngine.JarFile),
		"--set", fmt.Sprintf("sparkApiGateway.restUrl=%s", sparkApiGatewayUrl),
		"--set", fmt.Sprintf("sparkApiGateway.user=%s", apiGatewayUser),
		"--set", fmt.Sprintf("sparkApiGateway.password=%s", apiGatewayPassword),
	}

	err = kubelib.InstallHelm(commandEnvironment.Get(framework.CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvKyuubiHelmChart), kubeConfig, arguments, installName, installNamespace)
	return err
}
