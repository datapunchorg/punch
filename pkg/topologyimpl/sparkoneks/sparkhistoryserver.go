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

package sparkoneks

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	v1 "k8s.io/api/core/v1"
)

func DeployHistoryServer(commandEnvironment framework.CommandEnvironment, sparkComponentSpec SparkComponentSpec, region string, eksClusterName string) error {
	_, clientset, err := awslib.CreateKubernetesClient(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	err = InstallHistoryServerHelm(commandEnvironment, sparkComponentSpec, region, eksClusterName)
	if err != nil {
		return fmt.Errorf("failed to install Spark History Server helm chart: %s", err.Error())
	}

	helmInstallName := sparkComponentSpec.HistoryServer.HelmInstallName

	namespace := sparkComponentSpec.HistoryServer.Namespace
	podNamePrefix := helmInstallName
	err = kubelib.WaitPodsInPhases(clientset, namespace, podNamePrefix, []v1.PodPhase{v1.PodRunning})
	if err != nil {
		return fmt.Errorf("pod %s*** in namespace %s is not in phase %s", podNamePrefix, namespace, v1.PodRunning)
	}

	return nil
}

func InstallHistoryServerHelm(commandEnvironment framework.CommandEnvironment, sparkComponentSpec SparkComponentSpec, region string, eksClusterName string) error {
	// helm install spark-history-server third-party/helm-charts/spark-history-server/charts/spark-history-server --namespace spark-history-server --create-namespace

	kubeConfig, err := awslib.CreateKubeConfig(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := sparkComponentSpec.HistoryServer.HelmInstallName
	installNamespace := sparkComponentSpec.HistoryServer.Namespace

	arguments := []string{
		"--set", fmt.Sprintf("image.repository=%s", sparkComponentSpec.HistoryServer.ImageRepository),
		"--set", fmt.Sprintf("image.tag=%s", sparkComponentSpec.HistoryServer.ImageTag),
	}

	if !commandEnvironment.GetBoolOrElse(framework.CmdEnvWithMinikube, false) {
		arguments = append(arguments, "--set")
		arguments = append(arguments, fmt.Sprintf("sparkEventLogDir=%s", sparkComponentSpec.Gateway.SparkEventLogDir))
	}

	err = kubelib.InstallHelm(commandEnvironment.Get(framework.CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvHistoryServerHelmChart), kubeConfig, arguments, installName, installNamespace)
	return err
}
