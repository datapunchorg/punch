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

package gke

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/gcplib"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"github.com/datapunchorg/punch/pkg/resource"
	v1 "k8s.io/api/core/v1"
)

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*Topology)

	deployment := framework.NewDeployment()

	deployment.AddStep("createGkeCluster", "Create GKE cluster", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		err := resource.CreateGkeCluster(currentTopology.Spec.ProjectId, currentTopology.Spec.Location, currentTopology.Spec.GkeCluster)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		return framework.DeployableOutput{}, nil
	})

	if currentTopology.GetMetadata().GetCommandEnvironment().Get(CmdEnvNginxHelmChart) != "" {
		deployment.AddStep("deployNginxIngressController", "Deploy Nginx ingress controller", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
			return DeployNginxIngressController(*currentTopology.GetMetadata().GetCommandEnvironment(), currentTopology.Spec)
		})
	}

	err := deployment.Run()
	return deployment.GetOutput(), err
}

func DeployNginxIngressController(commandEnvironment framework.CommandEnvironment, topology TopologySpec) (map[string]interface{}, error) {
	nginxNamespace := topology.NginxIngress.Namespace
	helmInstallName := topology.NginxIngress.HelmInstallName
	nginxServiceName := DefaultNginxServiceName
	gkeClusterName := topology.GkeCluster.ClusterName
	kubeConfig, err := gcplib.CreateKubeConfig(commandEnvironment.Get(framework.CmdEnvKubeConfig), topology.ProjectId, topology.Location, gkeClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get kube config: %s", err.Error())
	}

	defer kubeConfig.Cleanup()

	arguments := []string{
		"--set", fmt.Sprintf("controller.service.enableHttp=%t", topology.NginxIngress.EnableHttp),
		"--set", fmt.Sprintf("controller.service.enableHttps=%t", topology.NginxIngress.EnableHttps),
	}

	if commandEnvironment.GetBoolOrElse(framework.CmdEnvWithMinikube, false) {
		arguments = append(arguments, "--set", "controller.service.type=NodePort")
		arguments = append(arguments, "--set", fmt.Sprintf("controller.service.nodePorts.http=%d", NodePortLocalHttp))
		arguments = append(arguments, "--set", fmt.Sprintf("controller.service.nodePorts.https=%d", NodePortLocalHttps))
	}

	kubelib.InstallHelm(commandEnvironment.Get(framework.CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvNginxHelmChart), kubeConfig, arguments, helmInstallName, nginxNamespace)

	_, clientset, err := gcplib.CreateKubernetesClient(commandEnvironment.Get(framework.CmdEnvKubeConfig), topology.ProjectId, topology.Location, gkeClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}
	err = kubelib.WaitPodsInPhases(clientset, nginxNamespace, nginxServiceName, []v1.PodPhase{v1.PodRunning})
	if err != nil {
		return nil, fmt.Errorf("pod %s*** in namespace %s is not in phase %s", nginxServiceName, nginxNamespace, v1.PodRunning)
	}

	urls, err := resource.GetGkeNginxLoadBalancerUrls(commandEnvironment, topology.ProjectId, topology.Location, gkeClusterName, nginxNamespace, nginxServiceName, NodePortLocalHttps)
	if err != nil {
		return nil, fmt.Errorf("failed to get NGINX load balancer urls: %s", err.Error())
	}

	output := make(map[string]interface{})
	output["loadBalancerUrls"] = urls

	preferredUrl := resource.GetLoadBalancerPreferredUrl(urls, topology.NginxIngress.EnableHttps && topology.NginxIngress.HttpsCertificate != "")
	output["loadBalancerPreferredUrl"] = preferredUrl

	return output, nil
}
