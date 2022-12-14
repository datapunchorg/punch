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
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/resource"
)

const (
	ToBeReplacedS3BucketName           = "todo_use_your_own_bucket_name"
	DefaultInstanceType1               = "t3.xlarge"
	DefaultInstanceType2               = "c5.xlarge"
	DefaultInstanceType3               = "r5.xlarge"
	DefaultNodeGroupSize               = 2
	DefaultMaxNodeGroupSize            = 10
	DefaultNginxIngressHelmInstallName = "ingress-nginx"
	DefaultNginxIngressNamespace       = "ingress-nginx"
	DefaultNginxEnableHttp             = true
	DefaultNginxEnableHttps            = true

	KindGkeTopology = "Gke"

	CmdEnvNginxHelmChart             = "nginxHelmChart"
	CmdEnvClusterAutoscalerHelmChart = "ClusterAutoscalerHelmChart"

	DefaultZone = "us-central1-c"
)

type Topology struct {
	framework.TopologyBase `json:",inline" yaml:",inline"`
	Spec                   TopologySpec `json:"spec"`
}

type TopologySpec struct {
	NamePrefix   string                   `json:"namePrefix" yaml:"namePrefix"`
	ProjectId    string                   `json:"projectId" yaml:"projectId"`
	Location     string                   `json:"location" yaml:"location"`
	VpcId        string                   `json:"vpcId" yaml:"vpcId"`
	S3BucketName string                   `json:"s3BucketName" yaml:"s3BucketName"`
	S3Policy     resource.IamPolicy       `json:"s3Policy" yaml:"s3Policy"`
	KafkaPolicy  resource.IamPolicy       `json:"kafkaPolicy" yaml:"kafkaPolicy"`
	GkeCluster   resource.GkeCluster      `json:"gkeCluster" yaml:"gkeCluster"`
	EksCluster   resource.EksCluster      `json:"eksCluster" yaml:"eksCluster"`
	NodeGroups   []resource.NodeGroup     `json:"nodeGroups" yaml:"nodeGroups"`
	NginxIngress NginxIngress             `json:"nginxIngress" yaml:"nginxIngress"`
	AutoScaling  resource.AutoScalingSpec `json:"autoScaling" yaml:"autoScaling"`
}

type NginxIngress struct {
	HelmInstallName string `json:"helmInstallName" yaml:"helmInstallName"`
	Namespace       string `json:"namespace" yaml:"namespace"`
	EnableHttp      bool   `json:"enableHttp" yaml:"enableHttp"`
	EnableHttps     bool   `json:"enableHttps" yaml:"enableHttps"`
}

func (t *Topology) GetKind() string {
	return t.Kind
}

func (t *Topology) GetMetadata() *framework.TopologyMetadata {
	return &t.Metadata
}
