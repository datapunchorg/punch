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
	"github.com/datapunchorg/punch/pkg/kubelib"
	"github.com/datapunchorg/punch/pkg/resource"
	"gopkg.in/yaml.v3"
)

type TopologyHandler struct {
}

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	topology := GenerateEksTopology()
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	result := CreateDefaultEksTopology(framework.DefaultNamePrefix, ToBeReplacedS3BucketName)
	err := yaml.Unmarshal(yamlContent, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML (%s): \n%s", err.Error(), string(yamlContent))
	}
	return &result, nil
}

func (t *TopologyHandler) Validate(topology framework.Topology, phase string) (framework.Topology, error) {
	currentTopology := topology.(*Topology)
	err := ValidateGkeTopologySpec(currentTopology.Spec, currentTopology.Metadata, phase)
	if err != nil {
		return nil, err
	}
	return topology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*Topology)

	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)

	deployment, err := CreateInstallDeployment(currentTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}

	err = deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*Topology)

	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)

	deployment, err := CreateUninstallDeployment(currentTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}

	err = deployment.Run()
	return deployment.GetOutput(), err
}

/*
func (t *TopologyHandler) PrintNotes(topology framework.Topology, deploymentOutput framework.DeploymentOutput) {
	currentTopology := topology.(*Topology)

	str := `
------------------------------
Example commands to use EKS cluster:
------------------------------
Step 1: run: aws eks update-kubeconfig --region %s --name %s
Step 2: run: kubectl get pods -A`
	log.Printf(str, currentTopology.Spec.Region, currentTopology.Spec.EksCluster.ClusterName)
}*/

func CreateInstallDeployment(topologySpec TopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment := framework.NewDeployment()

	if topologySpec.AutoScaling.EnableClusterAutoscaler && commandEnvironment.Get(CmdEnvClusterAutoscalerHelmChart) == "" {
		return framework.NewDeployment(), fmt.Errorf("please provide helm chart file location for Cluster Autoscaler")
	}

	err := kubelib.CheckHelm(commandEnvironment.Get(framework.CmdEnvHelmExecutable))
	if err != nil {
		return framework.NewDeployment(), err
	}

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
		err := resource.CreateGkeCluster(topologySpec.ProjectId, topologySpec.Location, topologySpec.GkeCluster)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		return framework.DeployableOutput{}, nil
		/*clusterSummary, err := resource.DescribeEksCluster(topologySpec.Region, topologySpec.EksCluster.ClusterName)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		return framework.DeployableOutput{"oidcIssuer": clusterSummary.OidcIssuer}, nil*/
	})

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

	if commandEnvironment.Get(CmdEnvNginxHelmChart) != "" {
		deployment.AddStep("deployNginxIngressController", "Deploy Nginx ingress controller", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
			return DeployNginxIngressController(commandEnvironment, topologySpec)
		})
	}*/

	return deployment, nil
}

func CreateUninstallDeployment(topologySpec TopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment := framework.NewDeployment()

	/*deployment.AddStep("deleteLoadBalancers", "Delete Load Balancers in EKS Cluster", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		err := awslib.DeleteAllLoadBalancersOnEks(topologySpec.Region, topologySpec.VpcId, topologySpec.EksCluster.ClusterName)
		return framework.NewDeploymentStepOutput(), err
	})
	deployment.AddStep("deleteOidcProvider", "Delete OIDC Provider", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		clusterSummary, err := resource.DescribeEksCluster(topologySpec.Region, topologySpec.EksCluster.ClusterName)
		if err != nil {
			log.Printf("[WARN] Cannot delete OIDC provider, failed to get EKS cluster %s in regsion %s: %s", topologySpec.EksCluster.ClusterName, topologySpec.Region, err.Error())
			return framework.NewDeploymentStepOutput(), nil
		}
		if clusterSummary.OidcIssuer != "" {
			log.Printf("Deleting OIDC Identity Provider %s", clusterSummary.OidcIssuer)
			err = awslib.DeleteOidcProvider(topologySpec.Region, clusterSummary.OidcIssuer)
			if err != nil {
				log.Printf("[WARN] Failed to delete OIDC provider %s: %s", clusterSummary.OidcIssuer, err.Error())
				return framework.NewDeploymentStepOutput(), nil
			}
			log.Printf("Deleted OIDC Identity Provider %s", clusterSummary.OidcIssuer)
		}
		return framework.NewDeploymentStepOutput(), nil
	})
	deployment.AddStep("deleteNodeGroups", "Delete Node Groups", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		for _, nodeGroup := range topologySpec.NodeGroups {
			err := awslib.DeleteNodeGroup(topologySpec.Region, topologySpec.EksCluster.ClusterName, nodeGroup.Name)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
		}
		return framework.NewDeploymentStepOutput(), nil
	}) */
	deployment.AddStep("deleteGkeCluster", "Delete GKE Cluster", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		resource.DeleteGkeCluster(topologySpec.ProjectId, topologySpec.Location, topologySpec.GkeCluster.ClusterName)
		return framework.NewDeploymentStepOutput(), nil
	})

	return deployment, nil
}
