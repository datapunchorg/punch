/*
Copyright 2022 DataPunch Project

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
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"k8s.io/api/core/v1"
	"log"
)

func DeployHistoryServer(commandEnvironment framework.CommandEnvironment, topology SparkOnEksTopologySpec) error {
	region := topology.EksSpec.Region
	clusterName := topology.EksSpec.Eks.ClusterName
	_, clientset, err := awslib.CreateKubernetesClient(region, commandEnvironment.Get(eks.CmdEnvKubeConfig), clusterName)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	err = InstallHistoryServerHelm(commandEnvironment, topology)
	if err != nil {
		return fmt.Errorf("failed to install Spark History Server helm chart: %s", err.Error())
	}

	helmInstallName := topology.HistoryServer.HelmInstallName

	namespace := topology.HistoryServer.Namespace
	podNamePrefix := helmInstallName
	err = kubelib.WaitPodsInPhases(clientset, namespace, podNamePrefix, []v1.PodPhase{v1.PodRunning})
	if err != nil {
		return fmt.Errorf("pod %s*** in namespace %s is not in phase %s", podNamePrefix, namespace, v1.PodRunning)
	}

	return nil
}

func InstallHistoryServerHelm(commandEnvironment framework.CommandEnvironment, topology SparkOnEksTopologySpec) error {
	// helm install spark-history-server third-party/helm-charts/spark-history-server/charts/spark-history-server --namespace spark-history-server --create-namespace

	kubeConfig, err := awslib.CreateKubeConfig(topology.EksSpec.Region, commandEnvironment.Get(eks.CmdEnvKubeConfig), topology.EksSpec.Eks.ClusterName)
	if err != nil {
		log.Fatalf("Failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := topology.HistoryServer.HelmInstallName
	installNamespace := topology.HistoryServer.Namespace

	arguments := []string{
		"--set", fmt.Sprintf("image.repository=%s", topology.HistoryServer.ImageRepository),
		"--set", fmt.Sprintf("image.tag=%s", topology.HistoryServer.ImageTag),
	}

	if !commandEnvironment.GetBoolOrElse(eks.CmdEnvWithMinikube, false) {
		arguments = append(arguments, "--set")
		arguments = append(arguments, fmt.Sprintf("sparkEventLogDir=%s", topology.ApiGateway.SparkEventLogDir))
	}

	kubelib.InstallHelm(commandEnvironment.Get(eks.CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvHistoryServerHelmChart), kubeConfig, arguments, installName, installNamespace)

	return nil
}

