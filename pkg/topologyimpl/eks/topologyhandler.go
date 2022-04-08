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

package eks

import (
	"bytes"
	"fmt"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"github.com/datapunchorg/punch/pkg/resource"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"strings"
	"text/template"
)

func init() {
	framework.DefaultTopologyHandlerManager.AddHandler(KindEksTopology, &TopologyHandler{})
}

type TopologyHandler struct {
}

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	topology := CreateEksTopologyTemplate()
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	result := CreateDefaultEksTopology(DefaultNamePrefix, ToBeReplacedS3BucketName)
	err := yaml.Unmarshal(yamlContent, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML (%s): \n%s", err.Error(), string(yamlContent))
	}
	return &result, nil
}

func (t *TopologyHandler) Resolve(topology framework.Topology, data framework.TemplateData) (framework.Topology, error) {
	topologyBytes, err := yaml.Marshal(topology)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal topology: %s", err.Error())
	}
	yamlContent := string(topologyBytes)

	tmpl, err := template.New("").Parse(yamlContent) // .Option("missingkey=error")?
	if err != nil {
		return nil, fmt.Errorf("failed to parse topology template (%s): %s", err.Error(), yamlContent)
	}

	templateData := framework.CreateTemplateDataWithRegion(data)

	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, &templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to execute topology template: %s", err.Error())
	}
	resolvedContent := buffer.String()
	resolvedTopology, err := t.Parse([]byte(resolvedContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse resolved topology (%s): %s", err.Error(), resolvedContent)
	}

	resolvedSpecificTopology := resolvedTopology.(*EksTopology)

	err = checkCmdEnvFolderExists(resolvedSpecificTopology.Metadata, CmdEnvNginxHelmChart)
	if err != nil {
		return nil, err
	}

	if resolvedSpecificTopology.Spec.AutoScaling.EnableClusterAutoscaler {
		err = checkCmdEnvFolderExists(resolvedSpecificTopology.Metadata, CmdEnvClusterAutoscalerHelmChart)
		if err != nil {
			return nil, err
		}
	}

	if resolvedSpecificTopology.Spec.AutoScaling.EnableClusterAutoscaler {
		err = awslib.CheckEksCtlCmd("eksctl")
		if err != nil {
			return nil, err
		}
	}

	return resolvedTopology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	specificTopology := topology.(*EksTopology)

	commandEnvironment := framework.CreateCommandEnvironment(specificTopology.Metadata.CommandEnvironment)

	deployment, err := BuildInstallDeployment(specificTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}

	err = deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	specificTopology := topology.(*EksTopology)

	commandEnvironment := framework.CreateCommandEnvironment(specificTopology.Metadata.CommandEnvironment)

	deployment, err := BuildUninstallDeployment(specificTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}

	err = deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) PrintUsageExample(topology framework.Topology, deploymentOutput framework.DeploymentOutput) {
	specificTopology := topology.(*EksTopology)

	str := `
------------------------------
Example commands to use EKS cluster:
------------------------------
Step 1: run: aws eks update-kubeconfig --region %s --name %s
Step 2: run: kubectl get pods -A`
	log.Printf(str, specificTopology.Spec.Region, specificTopology.Spec.Eks.ClusterName)
}

func BuildInstallDeployment(topologySpec EksTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment := framework.NewDeployment()

	if topologySpec.AutoScaling.EnableClusterAutoscaler && commandEnvironment.Get(CmdEnvClusterAutoscalerHelmChart) == "" {
		return framework.NewDeployment(), fmt.Errorf("please provide helm chart file location for Cluster Autoscaler")
	}

	kubelib.CheckHelmOrFatal(commandEnvironment.Get(CmdEnvHelmExecutable))
	if commandEnvironment.GetBoolOrElse(CmdEnvWithMinikube, false) {
		commandEnvironment.Set(CmdEnvKubeConfig, kubelib.GetKubeConfigPath())
		deployment.AddStep("minikubeProfile", "Set Minikube Profile", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("profile", topologySpec.Eks.ClusterName)
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("minikubeStart", "Start Minikube Cluster", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("start", "--memory", "4096") // TODO make memory size configurable
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("minikubeStatus", "Check Minikube Status", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("status")
			return framework.NewDeploymentStepOutput(), err
		})
	} else {
		deployment.AddStep("createS3Bucket", "Create S3 bucket", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			err := awslib.CreateS3Bucket(topologySpec.Region, topologySpec.S3BucketName)
			return framework.DeploymentStepOutput{"bucketName": topologySpec.S3BucketName}, err
		})

		deployment.AddStep("createInstanceIamRole", "Create Eks instance IAM role", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			roleName := CreateInstanceIamRole(topologySpec)
			return framework.DeploymentStepOutput{"roleName": roleName}, nil
		})

		deployment.AddStep("createEksCluster", "Create Eks cluster", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			err := resource.CreateEksCluster(topologySpec.Region, topologySpec.VpcId, topologySpec.Eks)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
			clusterSummary, err := resource.DescribeEksCluster(topologySpec.Region, topologySpec.Eks.ClusterName)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
			return framework.DeploymentStepOutput{"oidcIssuer": clusterSummary.OidcIssuer}, nil
		})

		deployment.AddStep("createNodeGroups", "Create node groups", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			stepOutput := c.GetStepOutput("createInstanceIamRole")
			roleName := stepOutput["roleName"].(string)
			if roleName == "" {
				return framework.NewDeploymentStepOutput(), fmt.Errorf("failed to get role name from previous step")
			}
			roleArn, err := awslib.GetIAMRoleArnByName(topologySpec.Region, roleName)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
			for _, nodeGroup := range topologySpec.NodeGroups {
				err := resource.CreateNodeGroup(topologySpec.Region, topologySpec.Eks.ClusterName, nodeGroup, roleArn)
				if err != nil {
					return framework.NewDeploymentStepOutput(), err
				}
			}
			return framework.NewDeploymentStepOutput(), nil
		})

		if topologySpec.AutoScaling.EnableClusterAutoscaler {
			deployment.AddStep("enableIamOidcProvider", "Enable IAM OIDC Provider", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
				awslib.RunEksCtlCmd("eksctl",
					[]string{"utils", "associate-iam-oidc-provider",
						"--region", topologySpec.Region,
						"--cluster", topologySpec.Eks.ClusterName,
						"--approve"})
				return framework.NewDeploymentStepOutput(), nil
			})

			deployment.AddStep("createClusterAutoscalerIAMRole", "Create Cluster Autoscaler IAM role", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
				oidcIssuer := c.GetStepOutput("createEksCluster")["oidcIssuer"].(string)
				idStr := "id/"
				index := strings.LastIndex(strings.ToLower(oidcIssuer), idStr)
				if index == -1 {
					return framework.NewDeploymentStepOutput(), fmt.Errorf("invalid OIDC issuer: %s", oidcIssuer)
				}
				oidcId := oidcIssuer[index+len(idStr):]
				roleName, err := CreateClusterAutoscalerIamRole(topologySpec, oidcId)
				return framework.DeploymentStepOutput{"roleName": roleName}, err
			})

			deployment.AddStep("createClusterAutoscalerIAMServiceAccount", "Create Cluster Autoscaler IAM service account", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
				roleName := c.GetStepOutput("createClusterAutoscalerIAMRole")["roleName"].(string)
				err := CreateClusterAutoscalerIamServiceAccount(commandEnvironment, topologySpec, roleName)
				return framework.NewDeploymentStepOutput(), err
			})

			deployment.AddStep("createClusterAutoscalerTagsOnNodeGroup", "Create Cluster Autoscaler tags on node groups", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
				for _, nodeGroup := range topologySpec.NodeGroups {
					err := awslib.CreateOrUpdateClusterAutoscalerTagsOnNodeGroup(topologySpec.Region, topologySpec.Eks.ClusterName, nodeGroup.Name)
					if err != nil {
						return framework.NewDeploymentStepOutput(), err
					}
				}
				return framework.NewDeploymentStepOutput(), nil
			})

			deployment.AddStep("deployClusterAutoscaler", "Deploy Cluster Autoscaler", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
				DeployClusterAutoscaler(commandEnvironment, topologySpec)
				return framework.NewDeploymentStepOutput(), nil
			})
		}
	}

	if commandEnvironment.Get(CmdEnvNginxHelmChart) != "" {
		deployment.AddStep("deployNginxIngressController", "Deploy Nginx ingress controller", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			return DeployNginxIngressController(commandEnvironment, topologySpec), nil
		})
	}

	return deployment, nil
}

func BuildUninstallDeployment(topologySpec EksTopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment := framework.NewDeployment()

	if commandEnvironment.GetBoolOrElse(CmdEnvWithMinikube, false) {
		deployment.AddStep("minikubeProfile", "Set Minikube Profile", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("profile", topologySpec.Eks.ClusterName)
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("minikubeStop", "Stop Minikube Cluster", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("stop")
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("minikubeDelete", "Delete Minikube Cluster", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("delete")
			return framework.NewDeploymentStepOutput(), err
		})
	} else {
		deployment.AddStep("deleteOidcProvider", "Delete OIDC Provider", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			clusterSummary, err := resource.DescribeEksCluster(topologySpec.Region, topologySpec.Eks.ClusterName)
			if err != nil {
				log.Printf("[WARN] Cannot delete OIDC provider, failed to get Eks cluster %s in regsion %s: %s", topologySpec.Eks.ClusterName, topologySpec.Region, err.Error())
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
		deployment.AddStep("deleteNodeGroups", "Delete Node Groups", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			for _, nodeGroup := range topologySpec.NodeGroups {
				err := awslib.DeleteNodeGroup(topologySpec.Region, topologySpec.Eks.ClusterName, nodeGroup.Name)
				if err != nil {
					return framework.NewDeploymentStepOutput(), err
				}
			}
			return framework.NewDeploymentStepOutput(), nil
		})

		deployment.AddStep("deleteLoadBalancer", "Delete Load Balancer", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			err := awslib.DeleteLoadBalancerOnEKS(topologySpec.Region, topologySpec.VpcId, topologySpec.Eks.ClusterName, "ingress-nginx")
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("deleteEksCluster", "Delete Eks Cluster", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			DeleteEksCluster(topologySpec.Region, topologySpec.Eks.ClusterName)
			return framework.NewDeploymentStepOutput(), nil
		})
	}
	return deployment, nil
}

func CreateEksTopologyTemplate() EksTopology {
	namePrefix := "{{ or .Values.namePrefix `my` }}"
	s3BucketName := "{{ or .Values.s3BucketName .DefaultS3BucketName }}"

	topology := CreateDefaultEksTopology(namePrefix, s3BucketName)

	topology.Spec.Region = "{{ or .Values.region `us-west-1` }}"
	topology.Spec.VpcId = "{{ or .Values.vpcId .DefaultVpcId }}"

	topology.Metadata.CommandEnvironment[CmdEnvHelmExecutable] = "{{ or .Env.helmExecutable `helm` }}"
	topology.Metadata.CommandEnvironment[CmdEnvWithMinikube] = "{{ or .Env.withMinikube `false` }}"
	topology.Metadata.CommandEnvironment[CmdEnvNginxHelmChart] = "{{ or .Env.nginxHelmChart `ingress-nginx/charts/ingress-nginx` }}"
	topology.Metadata.CommandEnvironment[CmdEnvClusterAutoscalerHelmChart] = "{{ or .Env.clusterAutoscalerHelmChart `cluster-autoscaler/charts/cluster-autoscaler` }}"
	topology.Metadata.CommandEnvironment[CmdEnvKubeConfig] = "{{ or .Env.kubeConfig `` }}"

	topology.Metadata.Notes["apiUserPassword"] = "Please make sure to provide API gateway user password when deploying the topology, e.g. --set apiUserPassword=your-password"

	return topology
}

func checkCmdEnvFolderExists(metadata framework.TopologyMetadata, cmdEnvKey string) error {
	cmdEnvValue := metadata.CommandEnvironment[cmdEnvKey]
	if cmdEnvValue == "" {
		return fmt.Errorf("Metadata.CommandEnvironment[\"%s\"] is empty", cmdEnvKey)
	}
	if _, err := os.Stat(cmdEnvValue); os.IsNotExist(err) {
		return fmt.Errorf("folder not exists (specified in Metadata.CommandEnvironment[\"%s\"]=\"%s\")", cmdEnvKey, cmdEnvValue)
	}
	return nil
}
