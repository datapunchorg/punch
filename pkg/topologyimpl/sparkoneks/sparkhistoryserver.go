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

package main

import (
	"fmt"
	"log"

	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	v1 "k8s.io/api/core/v1"
)

func DeployHistoryServer(commandEnvironment framework.CommandEnvironment, topology SparkOnEksTopologySpec) error {
	region := topology.Eks.Region
	clusterName := topology.Eks.EksCluster.ClusterName
	_, clientset, err := awslib.CreateKubernetesClient(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), clusterName)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	err = InstallHistoryServerHelm(commandEnvironment, topology)
	if err != nil {
		return fmt.Errorf("failed to install Spark History Server helm chart: %s", err.Error())
	}

	helmInstallName := topology.Spark.HistoryServer.HelmInstallName

	namespace := topology.Spark.HistoryServer.Namespace
	podNamePrefix := helmInstallName
	err = kubelib.WaitPodsInPhases(clientset, namespace, podNamePrefix, []v1.PodPhase{v1.PodRunning})
	if err != nil {
		return fmt.Errorf("pod %s*** in namespace %s is not in phase %s", podNamePrefix, namespace, v1.PodRunning)
	}

	return nil
}

func InstallHistoryServerHelm(commandEnvironment framework.CommandEnvironment, topology SparkOnEksTopologySpec) error {
	// helm install spark-history-server third-party/helm-charts/spark-history-server/charts/spark-history-server --namespace spark-history-server --create-namespace

	kubeConfig, err := awslib.CreateKubeConfig(topology.Eks.Region, commandEnvironment.Get(framework.CmdEnvKubeConfig), topology.Eks.EksCluster.ClusterName)
	if err != nil {
		log.Fatalf("Failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := topology.Spark.HistoryServer.HelmInstallName
	installNamespace := topology.Spark.HistoryServer.Namespace

	arguments := []string{
		"--set", fmt.Sprintf("image.repository=%s", topology.Spark.HistoryServer.ImageRepository),
		"--set", fmt.Sprintf("image.tag=%s", topology.Spark.HistoryServer.ImageTag),
	}

	if !commandEnvironment.GetBoolOrElse(framework.CmdEnvWithMinikube, false) {
		arguments = append(arguments, "--set")
		arguments = append(arguments, fmt.Sprintf("sparkEventLogDir=%s", topology.Spark.Gateway.SparkEventLogDir))
	}

	kubelib.InstallHelm(commandEnvironment.Get(framework.CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvHistoryServerHelmChart), kubeConfig, arguments, installName, installNamespace)

	return nil
}
