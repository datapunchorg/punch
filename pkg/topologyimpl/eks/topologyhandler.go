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
	topology := createEksTopologyTemplate()
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

	templateData := CreateEksTemplateData(data)

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

	deployment := framework.NewDeployment()

	commandEnvironment := framework.CreateCommandEnvironment(specificTopology.Metadata.CommandEnvironment)

	if specificTopology.Spec.AutoScaling.EnableClusterAutoscaler && commandEnvironment.Get(CmdEnvClusterAutoscalerHelmChart) == "" {
		return nil, fmt.Errorf("please provide helm chart file location for Cluster Autoscaler")
	}

	kubelib.CheckHelmOrFatal(commandEnvironment.Get(CmdEnvHelmExecutable))
	if commandEnvironment.GetBoolOrElse(CmdEnvWithMinikube, false) {
		commandEnvironment.Set(CmdEnvKubeConfig, kubelib.GetKubeConfigPath())
		deployment.AddStep("minikubeProfile", "Set Minikube Profile", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("profile", specificTopology.Spec.EKS.ClusterName)
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("minikubeStart", "Start Minikube Cluster", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("start", "--memory", "4096") // TODO make memory size configurable
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("minikubeStatus", "Check Minikube Status", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("status")
			return framework.NewDeploymentStepOutput(), err
		})
	} else {
		deployment.AddStep("createS3Bucket", "Create S3 bucket", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			specificTopology := t.(*EksTopology)
			err := awslib.CreateS3Bucket(specificTopology.Spec.Region, specificTopology.Spec.S3BucketName)
			return framework.DeploymentStepOutput{"bucketName": specificTopology.Spec.S3BucketName}, err
		})

		deployment.AddStep("createInstanceIAMRole", "Create EKS instance IAM role", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			specificTopology := t.(*EksTopology)
			roleName := CreateInstanceIAMRole(*specificTopology)
			return framework.DeploymentStepOutput{"roleName": roleName}, nil
		})

		deployment.AddStep("createEKSCluster", "Create EKS cluster", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			specificTopology := t.(*EksTopology)
			err := resource.CreateEksCluster(specificTopology.Spec.Region, specificTopology.Spec.VpcId, specificTopology.Spec.EKS)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
			clusterSummary, err := resource.DescribeEksCluster(specificTopology.Spec.Region, specificTopology.Spec.EKS.ClusterName)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
			return framework.DeploymentStepOutput{"oidcIssuer": clusterSummary.OidcIssuer}, nil
		})

		deployment.AddStep("createNodeGroups", "Create node groups", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			specificTopology := t.(*EksTopology)
			stepOutput := c.GetStepOutput("createInstanceIAMRole")
			roleName := stepOutput["roleName"].(string)
			if roleName == "" {
				return framework.NewDeploymentStepOutput(), fmt.Errorf("failed to get role name from previous step")
			}
			roleArn, err := awslib.GetIAMRoleArnByName(specificTopology.Spec.Region, roleName)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
			for _, nodeGroup := range specificTopology.Spec.NodeGroups {
				err := resource.CreateNodeGroup(specificTopology.Spec.Region, specificTopology.Spec.EKS.ClusterName, nodeGroup, roleArn)
				if err != nil {
					return framework.NewDeploymentStepOutput(), err
				}
			}
			return framework.NewDeploymentStepOutput(), nil
		})

		if specificTopology.Spec.AutoScaling.EnableClusterAutoscaler {
			deployment.AddStep("enableIamOidcProvider", "Enable IAM OIDC Provider", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
				specificTopology := t.(*EksTopology)
				awslib.RunEksCtlCmd("eksctl",
					[]string{"utils", "associate-iam-oidc-provider",
						"--region", specificTopology.Spec.Region,
						"--cluster", specificTopology.Spec.EKS.ClusterName,
						"--approve"})
				return framework.NewDeploymentStepOutput(), nil
			})

			deployment.AddStep("createClusterAutoscalerIAMRole", "Create Cluster Autoscaler IAM role", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
				specificTopology := t.(*EksTopology)
				oidcIssuer := c.GetStepOutput("createEKSCluster")["oidcIssuer"].(string)
				idStr := "id/"
				index := strings.LastIndex(strings.ToLower(oidcIssuer), idStr)
				if index == -1 {
					return framework.NewDeploymentStepOutput(), fmt.Errorf("invalid OIDC issuer: %s", oidcIssuer)
				}
				oidcId := oidcIssuer[index + len(idStr):]
				roleName, err := CreateClusterAutoscalerIAMRole(*specificTopology, oidcId)
				return framework.DeploymentStepOutput{"roleName": roleName}, err
			})

			deployment.AddStep("createClusterAutoscalerIAMServiceAccount", "Create Cluster Autoscaler IAM service account", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
				specificTopology := t.(*EksTopology)
				roleName := c.GetStepOutput("createClusterAutoscalerIAMRole")["roleName"].(string)
				err := CreateClusterAutoscalerIAMServiceAccount(commandEnvironment, *specificTopology, roleName)
				return framework.NewDeploymentStepOutput(), err
			})

			deployment.AddStep("createClusterAutoscalerTagsOnNodeGroup", "Create Cluster Autoscaler tags on node groups", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
				specificTopology := t.(*EksTopology)
				for _, nodeGroup := range specificTopology.Spec.NodeGroups {
					err := awslib.CreateOrUpdateClusterAutoscalerTagsOnNodeGroup(specificTopology.Spec.Region, specificTopology.Spec.EKS.ClusterName, nodeGroup.Name)
					if err != nil {
						return framework.NewDeploymentStepOutput(), err
					}
				}
				return framework.NewDeploymentStepOutput(), nil
			})

			deployment.AddStep("deployClusterAutoscaler", "Deploy Cluster Autoscaler", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
				specificTopology := t.(*EksTopology)
				DeployClusterAutoscaler(commandEnvironment, *specificTopology)
				return framework.NewDeploymentStepOutput(), nil
			})
		}
	}

	if commandEnvironment.Get(CmdEnvNginxHelmChart) != "" {
		deployment.AddStep("deployNginxIngressController", "Deploy Nginx ingress controller", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			specificTopology := t.(*EksTopology)
			return DeployNginxIngressController(commandEnvironment, *specificTopology), nil
		})
	}

	err := deployment.RunSteps(specificTopology)
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	specificTopology := topology.(*EksTopology)

	commandEnvironment := framework.CreateCommandEnvironment(specificTopology.Metadata.CommandEnvironment)
	deployment := framework.NewDeployment()

	if commandEnvironment.GetBoolOrElse(CmdEnvWithMinikube, false) {
		deployment.AddStep("minikubeProfile", "Set Minikube Profile", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("profile", specificTopology.Spec.EKS.ClusterName)
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("minikubeStop", "Stop Minikube Cluster", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("stop")
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("minikubeDelete", "Delete Minikube Cluster", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			_, err := resource.MinikubeExec("delete")
			return framework.NewDeploymentStepOutput(), err
		})
	} else {
		deployment.AddStep("deleteOidcProvider", "Delete OIDC Provider", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			specificTopology := t.(*EksTopology)
			clusterSummary, err := resource.DescribeEksCluster(specificTopology.Spec.Region, specificTopology.Spec.EKS.ClusterName)
			if err != nil {
				log.Printf("[WARN] Cannot delete OIDC provider, failed to get EKS cluster %s in regsion %s: %s", specificTopology.Spec.EKS.ClusterName, specificTopology.Spec.Region, err.Error())
				return framework.NewDeploymentStepOutput(), nil
			}
			if clusterSummary.OidcIssuer != "" {
				log.Printf("Deleting OIDC Identity Provider %s", clusterSummary.OidcIssuer)
				err = awslib.DeleteOidcProvider(specificTopology.Spec.Region, clusterSummary.OidcIssuer)
				if err != nil {
					log.Printf("[WARN] Failed to delete OIDC provider %s: %s", clusterSummary.OidcIssuer, err.Error())
					return framework.NewDeploymentStepOutput(), nil
				}
				log.Printf("Deleted OIDC Identity Provider %s", clusterSummary.OidcIssuer)
			}
			return framework.NewDeploymentStepOutput(), nil
		})
		deployment.AddStep("deleteNodeGroups", "Delete Node Groups", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			specificTopology := t.(*EksTopology)
			for _, nodeGroup := range specificTopology.Spec.NodeGroups {
				err := awslib.DeleteNodeGroup(specificTopology.Spec.Region, specificTopology.Spec.EKS.ClusterName, nodeGroup.Name)
				if err != nil {
					return framework.NewDeploymentStepOutput(), err
				}
			}
			return framework.NewDeploymentStepOutput(), nil
		})

		deployment.AddStep("deleteLoadBalancer", "Delete Load Balancer", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			specificTopology := t.(*EksTopology)
			err := awslib.DeleteLoadBalancerOnEKS(specificTopology.Spec.Region, specificTopology.Spec.VpcId, specificTopology.Spec.EKS.ClusterName, "ingress-nginx")
			return framework.NewDeploymentStepOutput(), err
		})

		deployment.AddStep("deleteEKSCluster", "Delete EKS Cluster", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			specificTopology := t.(*EksTopology)
			DeleteEKSCluster(specificTopology.Spec.Region, specificTopology.Spec.EKS.ClusterName)
			return framework.NewDeploymentStepOutput(), nil
		})
	}

	err := deployment.RunSteps(specificTopology)
	return deployment.GetOutput(), err
}

func createEksTopologyTemplate() EksTopology {
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
