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
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/resource"
	"gopkg.in/yaml.v3"
)

const (
	ToBeReplacedS3BucketName           = "todo_use_your_own_bucket_name"
	DefaultInstanceType = "t3.medium"
	DefaultNodeGroupSize = 3
	DefaultNginxIngressHelmInstallName = "ingress-nginx"
	DefaultNginxIngressNamespace = "ingress-nginx"
	DefaultNginxEnableHttp = true
	DefaultNginxEnableHttps = true

	KindEksTopology = "Eks"

	CmdEnvHelmExecutable = "helmExecutable"
	CmdEnvNginxHelmChart = "nginxHelmChart"
	CmdEnvKubeConfig = "kubeConfig"

	DefaultVersion = "datapunch.org/v1alpha1"
	DefaultRegion = "us-west-1"
	DefaultNamePrefix = "my"
	DefaultHelmExecutable = "helm"
)

type EksTopology struct {
	ApiVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind     string            `json:"kind" yaml:"kind"`
	Metadata framework.TopologyMetadata  `json:"metadata"`
	Spec     EksTopologySpec `json:"spec"`
}

type EksTopologySpec struct {
	NamePrefix string `json:"namePrefix" yaml:"namePrefix"`
	Region string `json:"region"`
	VpcId string `json:"vpcId" yaml:"vpcId"`
	S3BucketName string                   `json:"s3BucketName" yaml:"s3BucketName"`
	S3Policy resource.IAMPolicy `json:"s3Policy" yaml:"s3Policy"`
	EKS           resource.EKSCluster  `json:"eks" yaml:"eks"`
	NodeGroups    []resource.NodeGroup `json:"nodeGroups" yaml:"nodeGroups"`
	NginxIngress  NginxIngress             `json:"nginxIngress" yaml:"nginxIngress"`
}

type SparkApiGateway struct {
	UserName     string `json:"userName" yaml:"userName"`
	UserPassword string `json:"userPassword" yaml:"userPassword"`
}

type SparkOperator struct {
	HelmInstallName string `json:"helmInstallName" yaml:"helmInstallName"`
	Namespace string `json:"namespace" yaml:"namespace"`
	ImageRepository string `json:"imageRepository" yaml:"imageRepository"`
	ImageTag string `json:"imageTag" yaml:"imageTag"`
	SparkApplicationNamespace string `json:"sparkApplicationNamespace" yaml:"sparkApplicationNamespace"`
}

type NginxIngress struct {
	HelmInstallName string `json:"helmInstallName" yaml:"helmInstallName"`
	Namespace string `json:"namespace" yaml:"namespace"`
	EnableHttp bool `json:"enableHttp" yaml:"enableHttp"`
	EnableHttps bool `json:"enableHttps" yaml:"enableHttps"`
}

func CreateDefaultEksTopology(namePrefix string, s3BucketName string) EksTopology {
	topologyName := fmt.Sprintf("%s-spark-k8s", namePrefix)
	k8sClusterName := fmt.Sprintf("%s-k8s-01", namePrefix)
	controlPlaneRoleName := fmt.Sprintf("%s-eks-control-plane", namePrefix)
	instanceRoleName := fmt.Sprintf("%s-eks-instance", namePrefix)
	securityGroupName := fmt.Sprintf("%s-sg-01", namePrefix)
	nodeGroupName := fmt.Sprintf("%s-ng-01", k8sClusterName)
	topology := EksTopology{
		ApiVersion: DefaultVersion,
		Kind:       KindEksTopology,
		Metadata: framework.TopologyMetadata{
			Name: topologyName,
			CommandEnvironment: map[string]string{
				CmdEnvHelmExecutable: DefaultHelmExecutable,
			},
			Notes: map[string]string{},
		},
		Spec: EksTopologySpec{
			NamePrefix: namePrefix,
			Region:     DefaultRegion,
			VpcId: "",
			S3BucketName: s3BucketName,
			S3Policy: resource.IAMPolicy{
			},
			EKS: resource.EKSCluster{
				ClusterName:          k8sClusterName,
				ControlPlaneRole: resource.IAMRole{
					Name:                 controlPlaneRoleName,
					AssumeRolePolicyDocument: framework.DefaultEKSAssumeRolePolicyDocument,
					ExtraPolicyArns: []string {
						"arn:aws:iam::aws:policy/AmazonEKSClusterPolicy",
					},
				},
				InstanceRole: resource.IAMRole{
					Name:                     instanceRoleName,
					AssumeRolePolicyDocument: framework.DefaultEC2AssumeRolePolicyDocument,
					ExtraPolicyArns: []string {
						"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
						"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
						"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
						"arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
					},
				},
				SecurityGroups: []resource.SecurityGroup{
					{
						Name: securityGroupName,
						InboundRules: []resource.SecurityGroupInboundRule{
							{
								IPProtocol: "-1",
								FromPort: -1,
								ToPort: -1,
								IPRanges: []string{"0.0.0.0/0"},
							},
						},
					},
				},
			},
			NodeGroups: []resource.NodeGroup{
				{
					Name:          nodeGroupName,
					InstanceTypes: []string{DefaultInstanceType},
					DesiredSize:   DefaultNodeGroupSize,
					MaxSize:       DefaultNodeGroupSize,
					MinSize:       DefaultNodeGroupSize,
				},
			},
			NginxIngress: NginxIngress{
				HelmInstallName: DefaultNginxIngressHelmInstallName,
				Namespace:       DefaultNginxIngressNamespace,
				EnableHttp:      DefaultNginxEnableHttp,
				EnableHttps:     DefaultNginxEnableHttps,
			},
		},
	}
	UpdateSparkTopologyByS3BucketName(&topology, s3BucketName)
	return topology
}

func UpdateSparkTopologyByS3BucketName(topology *EksTopology, s3BucketName string) {
	topology.Spec.S3BucketName = s3BucketName
	topology.Spec.S3Policy.Name = fmt.Sprintf("%s-s3", s3BucketName)
	topology.Spec.S3Policy.PolicyDocument = fmt.Sprintf(`{"Version":"2012-10-17","Statement":[
{"Effect":"Allow","Action":"s3:*","Resource":["arn:aws:s3:::%s", "arn:aws:s3:::%s/*"]}
]}`, s3BucketName, s3BucketName)
}

func (t *EksTopology) GetKind() string {
	return t.Kind
}

func (t *EksTopology) ToString() string {
	topologyBytes, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology: %s)", err.Error())
	}
	return string(topologyBytes)
}
