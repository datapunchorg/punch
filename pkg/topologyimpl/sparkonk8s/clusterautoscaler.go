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

package sparkonk8s

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	v1 "k8s.io/api/core/v1"
	"log"
)

func DeployClusterAutoscaler(commandEnvironment framework.CommandEnvironment, topology SparkTopology) {
	InstallClusterAutoscalerHelm(commandEnvironment, topology)
}

func InstallClusterAutoscalerHelm(commandEnvironment framework.CommandEnvironment, topology SparkTopology) {
	// helm install cluster-autoscaler third-party/helm-charts/cluster-autoscaler --set autoDiscovery.clusterName=my-k8s-01 --set awsRegion=us-west-1

	kubeConfig, err := awslib.CreateKubeConfig(topology.Spec.Region, commandEnvironment.Get(CmdEnvKubeConfig), topology.Spec.EKS.ClusterName)
	if err != nil {
		log.Fatalf("Failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := "cluster-autoscaler"
	installNamespace := "kube-system"

	arguments := []string{
		"--set", fmt.Sprintf("awsRegion=%s", topology.Spec.Region),
		"--set", fmt.Sprintf("autoDiscovery.clusterName=%s", topology.Spec.EKS.ClusterName),
		"--set", "cloudProvider=aws",
		"--set", "rbac.serviceAccount.create=false",
		"--set", "rbac.serviceAccount.name=cluster-autoscaler",
	}

	kubelib.InstallHelm(commandEnvironment.Get(CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvClusterAutoscalerHelmChart), kubeConfig, arguments, installName, installNamespace)

	region := topology.Spec.Region
	clusterName := topology.Spec.EKS.ClusterName

	_, clientset, err := awslib.CreateKubernetesClient(region, commandEnvironment.Get(CmdEnvKubeConfig), clusterName)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %s", err.Error())
	}

	podNamePrefix := "cluster-autoscaler-aws-cluster-autoscaler"
	err = kubelib.WaitPodsInPhase(clientset, installNamespace, podNamePrefix, v1.PodRunning)
	if err != nil {
		log.Fatalf("Pod %s*** in namespace %s is not in phase %s", podNamePrefix, installNamespace, v1.PodRunning)
	}
}
