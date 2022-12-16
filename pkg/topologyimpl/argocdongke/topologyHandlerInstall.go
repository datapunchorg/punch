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

package argocdongke

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/gcplib"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"github.com/datapunchorg/punch/pkg/resource"
	"github.com/datapunchorg/punch/pkg/topologyimpl/gke"
	v1 "k8s.io/api/core/v1"
	"log"
	"strings"
)

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*Topology)

	deployment := framework.NewDeployment()

	deployment.AddStep("createGkeCluster", "Create GKE cluster", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		err := resource.CreateGkeCluster(currentTopology.Spec.GkeSpec.ProjectId, currentTopology.Spec.GkeSpec.Location, currentTopology.Spec.GkeSpec.GkeCluster)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		return framework.DeployableOutput{}, nil
	})

	deployment.AddStep("deployNginxIngressController", "Deploy Nginx ingress controller", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		return gke.DeployNginxIngressController(*currentTopology.GetMetadata().GetCommandEnvironment(), currentTopology.Spec.GkeSpec)
	})

	deployment.AddStep("deployArgocd", "Deploy Argocd", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		return DeployArgocd(*currentTopology.GetMetadata().GetCommandEnvironment(), currentTopology.Spec)
	})

	err := deployment.Run()
	return deployment.GetOutput(), err
}

func DeployArgocd(commandEnvironment framework.CommandEnvironment, topology TopologySpec) (map[string]interface{}, error) {
	kubeConfig, err := gcplib.CreateKubeConfig(commandEnvironment.Get(framework.CmdEnvKubeConfig), topology.GkeSpec.ProjectId, topology.GkeSpec.Location, topology.GkeSpec.GkeCluster.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get kube config: %s", err.Error())
	}
	defer kubeConfig.Cleanup()

	namespace := "argocd"

	{
		cmd := fmt.Sprintf("create namespace %s", namespace)
		arguments := strings.Split(cmd, " ")
		err = kubelib.RunKubectlWithKubeConfig(commandEnvironment.Get(framework.CmdEnvKubectlExecutable), kubeConfig, arguments)
		if err != nil {
			log.Printf("Failed to run kubectl %s: %s", cmd, err.Error())
		}
	}
	{
		cmd := fmt.Sprintf("apply -n %s -f %s", namespace, topology.ArgocdInstallYaml)
		arguments := strings.Split(cmd, " ")
		err = kubelib.RunKubectlWithKubeConfig(commandEnvironment.Get(framework.CmdEnvKubectlExecutable), kubeConfig, arguments)
		if err != nil {
			return nil, err
		}
	}

	_, clientset, err := gcplib.CreateKubernetesClient(commandEnvironment.Get(framework.CmdEnvKubeConfig), topology.GkeSpec.ProjectId, topology.GkeSpec.Location, topology.GkeSpec.GkeCluster.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	podNamePrefix := "argocd-server"
	err = kubelib.WaitPodsInPhases(clientset, namespace, podNamePrefix, []v1.PodPhase{v1.PodRunning})
	if err != nil {
		return nil, fmt.Errorf("pod %s*** in namespace %s is not in phase %s", podNamePrefix, namespace, v1.PodRunning)
	}

	output := make(map[string]interface{})

	return output, nil
}
