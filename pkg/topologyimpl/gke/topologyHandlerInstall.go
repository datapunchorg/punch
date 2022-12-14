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

	/*deployment.AddStep("createS3Bucket", "Create S3 bucket", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		err := awslib.CreateS3Bucket(topologySpec.Region, topologySpec.S3BucketName)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		return framework.DeployableOutput{"bucketName": topologySpec.S3BucketName}, nil
	})

	deployment.AddStep("createInstanceIamRole", "Create EKS instance IAM role", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		roleName, err := CreateInstanceIamRole(topologySpec)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		return framework.DeployableOutput{"roleName": roleName}, err
	})*/

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

	/*deployment.AddStep("createNodeGroups", "Create node groups", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		stepOutput := c.GetStepOutput("createInstanceIamRole")
		roleName := stepOutput["roleName"].(string)
		if roleName == "" {
			return framework.NewDeploymentStepOutput(), fmt.Errorf("failed to get role name from previous step")
		}
		roleArn, err := awslib.GetIamRoleArnByName(topologySpec.Region, roleName)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		for _, nodeGroup := range topologySpec.NodeGroups {
			err := resource.CreateNodeGroup(topologySpec.Region, topologySpec.EksCluster.ClusterName, nodeGroup, roleArn)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
		}
		return framework.NewDeploymentStepOutput(), nil
	})

	if topologySpec.AutoScaling.EnableClusterAutoscaler {
		deployment.AddStep("enableIamOidcProvider", "Enable IAM OIDC Provider", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
			awslib.RunEksCtlCmd("eksctl",
				[]string{"utils", "associate-iam-oidc-provider",
					"--region", topologySpec.Region,
					"--cluster", topologySpec.EksCluster.ClusterName,
					"--approve"})
			return framework.NewDeploymentStepOutput(), nil
		})

		deployment.AddStep("createClusterAutoscalerIamRole", "Create Cluster Autoscaler IAM role", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
			oidcIssuer := c.GetStepOutput("createEksCluster")["oidcIssuer"].(string)
			idStr := "id/"
			index := strings.LastIndex(strings.ToLower(oidcIssuer), idStr)
			if index == -1 {
				return framework.NewDeploymentStepOutput(), fmt.Errorf("invalid OIDC issuer: %s", oidcIssuer)
			}
			oidcId := oidcIssuer[index+len(idStr):]
			roleName, err := CreateClusterAutoscalerIamRole(topologySpec, oidcId)
			return framework.DeployableOutput{"roleName": roleName}, err
		})

		deployment.AddStep("createClusterAutoscalerIAMServiceAccount", "Create Cluster Autoscaler IAM service account", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
			roleName := c.GetStepOutput("createClusterAutoscalerIamRole")["roleName"].(string)
			err := CreateClusterAutoscalerIamServiceAccount(commandEnvironment, topologySpec, roleName)
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("createClusterAutoscalerTagsOnNodeGroup", "Create Cluster Autoscaler tags on node groups", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
			for _, nodeGroup := range topologySpec.NodeGroups {
				err := awslib.CreateOrUpdateClusterAutoscalerTagsOnNodeGroup(topologySpec.Region, topologySpec.EksCluster.ClusterName, nodeGroup.Name)
				if err != nil {
					return framework.NewDeploymentStepOutput(), err
				}
			}
			return framework.NewDeploymentStepOutput(), nil
		})

		deployment.AddStep("deployClusterAutoscaler", "Deploy Cluster Autoscaler", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
			err := InstallClusterAutoscalerHelm(commandEnvironment, topologySpec)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
			return framework.NewDeploymentStepOutput(), nil
		})
	}
	*/

	err := deployment.Run()
	return deployment.GetOutput(), err
}

func DeployNginxIngressController(commandEnvironment framework.CommandEnvironment, topology TopologySpec) (map[string]interface{}, error) {
	nginxNamespace := topology.NginxIngress.Namespace
	helmInstallName := topology.NginxIngress.HelmInstallName
	nginxServiceName := DefaultNginxServiceName
	eksClusterName := topology.GkeCluster.ClusterName
	kubeConfig, err := gcplib.CreateKubeConfig(commandEnvironment.Get(framework.CmdEnvKubeConfig), topology.ProjectId, topology.Location, eksClusterName)
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

	_, clientset, err := gcplib.CreateKubernetesClient(commandEnvironment.Get(framework.CmdEnvKubeConfig), topology.ProjectId, topology.Location, eksClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}
	err = kubelib.WaitPodsInPhases(clientset, nginxNamespace, nginxServiceName, []v1.PodPhase{v1.PodRunning})
	if err != nil {
		return nil, fmt.Errorf("pod %s*** in namespace %s is not in phase %s", nginxServiceName, nginxNamespace, v1.PodRunning)
	}

	//urls, err := resource.GetEksNginxLoadBalancerUrls(commandEnvironment, region, eksClusterName, nginxNamespace, nginxServiceName, NodePortLocalHttps)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to get NGINX load balancer urls: %s", err.Error())
	//}

	output := make(map[string]interface{})
	//output["loadBalancerUrls"] = urls

	//preferredUrl := resource.GetLoadBalancerPreferredUrl(urls)
	//output["loadBalancerPreferredUrl"] = preferredUrl

	return output, nil
}
