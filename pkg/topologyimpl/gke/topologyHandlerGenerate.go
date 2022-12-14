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
	"github.com/datapunchorg/punch/pkg/resource"
)

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	namePrefix := framework.DefaultNamePrefixTemplate
	s3BucketName := framework.DefaultS3BucketNameTemplate

	topologyName := fmt.Sprintf("%s-eks-01", namePrefix)
	gkeClusterName := topologyName
	controlPlaneRoleName := fmt.Sprintf("%s-eks-control-plane", namePrefix)
	instanceRoleName := fmt.Sprintf("%s-eks-instance", namePrefix)
	securityGroupName := fmt.Sprintf("%s-eks-sg-01", namePrefix)
	nodeGroupName := fmt.Sprintf("%s-ng-01", gkeClusterName)
	topology := Topology{
		TopologyBase: framework.TopologyBase{
			ApiVersion: framework.DefaultVersion,
			Kind:       KindGkeTopology,
			Metadata: framework.TopologyMetadata{
				Name: topologyName,
				CommandEnvironment: map[string]string{
					framework.CmdEnvHelmExecutable:    framework.DefaultHelmExecutable,
					framework.CmdEnvKubectlExecutable: framework.DefaultKubectlExecutable,
					framework.CmdEnvWithMinikube:      "false",
					CmdEnvNginxHelmChart:              "third-party/helm-charts/ingress-nginx/charts/ingress-nginx",
					CmdEnvClusterAutoscalerHelmChart:  "third-party/helm-charts/cluster-autoscaler/charts/cluster-autoscaler",
					framework.CmdEnvKubeConfig:        "",
				},
				Notes: map[string]string{},
			},
		},
		Spec: TopologySpec{
			NamePrefix:   namePrefix,
			ProjectId:    fmt.Sprintf("{{ .Values.projectId }}"),
			Location:     fmt.Sprintf("{{ or .Values.location `%s` }}", DefaultZone),
			VpcId:        "{{ or .Values.vpcId .DefaultVpcId }}",
			S3BucketName: s3BucketName,
			S3Policy: resource.IamPolicy{
				Name: fmt.Sprintf("%s-s3", s3BucketName),
				PolicyDocument: fmt.Sprintf(`{"Version":"2012-10-17","Statement":[
{"Effect":"Allow","Action":"s3:*","Resource":["arn:aws:s3:::%s", "arn:aws:s3:::%s/*"]}
]}`, s3BucketName, s3BucketName),
			},
			KafkaPolicy: resource.IamPolicy{
				Name: fmt.Sprintf("%s-eks-kafka-cluster", namePrefix),
				PolicyDocument: `{"Version":"2012-10-17","Statement":[
{"Effect":"Allow","Action":"kafka-cluster:*","Resource":"*"}
]}`,
			},
			GkeCluster: resource.GkeCluster{
				ClusterName: gkeClusterName,
			},
			EksCluster: resource.EksCluster{
				ClusterName: gkeClusterName,
				EksVersion:  "1.21",
				// TODO fill in default value for SubnetIds
				ControlPlaneRole: resource.IamRole{
					Name:                     controlPlaneRoleName,
					AssumeRolePolicyDocument: framework.DefaultEKSAssumeRolePolicyDocument,
					ExtraPolicyArns: []string{
						"arn:aws:iam::aws:policy/AmazonEKSClusterPolicy",
					},
				},
				InstanceRole: resource.IamRole{
					Name:                     instanceRoleName,
					AssumeRolePolicyDocument: framework.DefaultEC2AssumeRolePolicyDocument,
					ExtraPolicyArns: []string{
						"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
						"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
						"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
						"arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
						"arn:aws:iam::aws:policy/AmazonMSKFullAccess",
					},
				},
				SecurityGroups: []resource.SecurityGroup{
					{
						Name: securityGroupName,
						InboundRules: []resource.SecurityGroupInboundRule{
							{
								IPProtocol: "-1",
								FromPort:   -1,
								ToPort:     -1,
								IPRanges:   []string{"{{ or .Values.vpcCidrBlock .DefaultVpcCidrBlock }}"},
							},
						},
					},
				},
			},
			AutoScaling: resource.AutoScalingSpec{
				EnableClusterAutoscaler: false,
				ClusterAutoscalerIamRole: resource.IamRole{
					Name: fmt.Sprintf("%s-cluster-autoscaler-role", namePrefix),
					Policies: []resource.IamPolicy{
						resource.IamPolicy{
							Name: fmt.Sprintf("%s-cluster-autoscaler-policy", namePrefix),
							PolicyDocument: `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": [
                "autoscaling:DescribeAutoScalingGroups",
                "autoscaling:DescribeAutoScalingInstances",
                "autoscaling:DescribeLaunchConfigurations",
                "autoscaling:DescribeTags",
                "autoscaling:SetDesiredCapacity",
                "autoscaling:TerminateInstanceInAutoScalingGroup",
                "ec2:DescribeLaunchTemplateVersions",
                "ec2:DescribeInstanceTypes"
            ],
            "Resource": "*",
            "Effect": "Allow"
        }
    ]
}`,
						},
					},
				},
			},
			NodeGroups: []resource.NodeGroup{
				{
					Name:          nodeGroupName,
					InstanceTypes: []string{DefaultInstanceType1, DefaultInstanceType2, DefaultInstanceType3},
					DesiredSize:   DefaultNodeGroupSize,
					MaxSize:       DefaultMaxNodeGroupSize,
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
	return &topology, nil
}
