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
	"os"
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

	sparkData := CreateEksTemplateData(data)

	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, &sparkData)
	if err != nil {
		return nil, fmt.Errorf("failed to execute topology template: %s", err.Error())
	}
	resolvedContent := buffer.String()
	resolvedTopology, err := t.Parse([]byte(resolvedContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse resolved topology (%s): %s", err.Error(), resolvedContent)
	}

	resolvedSparkTopology := resolvedTopology.(*EksTopology)

	err = checkCmdEnvFolderExists(resolvedSparkTopology.Metadata, CmdEnvNginxHelmChart)
	if err != nil {
		return nil, err
	}

	return resolvedTopology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	sparkTopology := topology.(*EksTopology)

	deployment := framework.NewDeployment()

	commandEnvironment := framework.CreateCommandEnvironment(sparkTopology.Metadata.CommandEnvironment)

	kubelib.CheckHelmOrFatal(commandEnvironment.Get(CmdEnvHelmExecutable))

	deployment.AddStep("createS3Bucket", "Create S3 bucket", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
		sparkTopology := t.(*EksTopology)
		err := awslib.CreateS3Bucket(sparkTopology.Spec.Region, sparkTopology.Spec.S3BucketName)
		return framework.NewDeploymentStepOutput(), err
	})

	deployment.AddStep("createInstanceIAMRole", "Create EKS instance IAM role", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
		sparkTopology := t.(*EksTopology)
		roleName := CreateInstanceIAMRole(*sparkTopology)
		return framework.DeploymentStepOutput{"roleName": roleName}, nil
	})

	deployment.AddStep("createEKSCluster", "Create EKS cluster", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
		sparkTopology := t.(*EksTopology)
		err := resource.CreateEksCluster(sparkTopology.Spec.Region, sparkTopology.Spec.VpcId, sparkTopology.Spec.EKS)
		return framework.NewDeploymentStepOutput(), err
	})

	deployment.AddStep("createNodeGroups", "Create node groups", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
		sparkTopology := t.(*EksTopology)
		stepOutput := c.GetStepOutput("createInstanceIAMRole")
		roleName := stepOutput["roleName"].(string)
		if roleName == "" {
			return framework.NewDeploymentStepOutput(), fmt.Errorf("failed to get role name from previous step")
		}
		roleArn, err := awslib.GetIAMRoleArnByName(sparkTopology.Spec.Region, roleName)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		for _, nodeGroup := range sparkTopology.Spec.NodeGroups {
			err := resource.CreateNodeGroup(sparkTopology.Spec.Region, sparkTopology.Spec.EKS.ClusterName, nodeGroup, roleArn)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
		}
		return framework.NewDeploymentStepOutput(), nil
	})

	if commandEnvironment.Get(CmdEnvNginxHelmChart) != "" {
		deployment.AddStep("deployNginxIngressController", "Deploy Nginx ingress controller", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
			sparkTopology := t.(*EksTopology)
			return DeployNginxIngressController(commandEnvironment, *sparkTopology), nil
		})
	}

	err := deployment.RunSteps(sparkTopology)
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	sparkTopology := topology.(*EksTopology)

	deployment := framework.NewDeployment()

	deployment.AddStep("deleteNodeGroups", "Delete Node Groups", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
		sparkTopology := t.(*EksTopology)
		for _, nodeGroup := range sparkTopology.Spec.NodeGroups {
			err := awslib.DeleteNodeGroup(sparkTopology.Spec.Region, sparkTopology.Spec.EKS.ClusterName, nodeGroup.Name)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
		}
		return framework.NewDeploymentStepOutput(), nil
	})

	deployment.AddStep("deleteLoadBalancer", "Delete Load Balancer", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
		sparkTopology := t.(*EksTopology)
		err := awslib.DeleteLoadBalancerOnEKS(sparkTopology.Spec.Region, sparkTopology.Spec.VpcId, sparkTopology.Spec.EKS.ClusterName, "ingress-nginx")
		return framework.NewDeploymentStepOutput(), err
	})

	deployment.AddStep("deleteEKSCluster", "Delete EKS Cluster", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
		sparkTopology := t.(*EksTopology)
		err := awslib.DeleteEKSCluster(sparkTopology.Spec.Region, sparkTopology.Spec.EKS.ClusterName)
		return framework.NewDeploymentStepOutput(), err
	})

	err := deployment.RunSteps(sparkTopology)
	return deployment.GetOutput(), err
}

func createEksTopologyTemplate() EksTopology {
	namePrefix := "{{ or .Values.namePrefix `my` }}"
	s3BucketName := "{{ or .Values.s3BucketName .DefaultS3BucketName }}"

	topology := CreateDefaultEksTopology(namePrefix, s3BucketName)

	topology.Spec.Region = "{{ or .Values.region `us-west-1` }}"
	topology.Spec.VpcId = "{{ or .Values.vpcId .DefaultVpcId }}"

	topology.Metadata.CommandEnvironment[CmdEnvHelmExecutable] = "{{ or .Env.helmExecutable `helm` }}"
	topology.Metadata.CommandEnvironment[CmdEnvNginxHelmChart] = "{{ or .Env.nginxHelmChart `ingress-nginx/charts/ingress-nginx` }}"
	topology.Metadata.CommandEnvironment[CmdEnvKubeConfig] = "{{ or .Env.kubeConfig `` }}"

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
