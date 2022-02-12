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

package ec2

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/resource"
	"gopkg.in/yaml.v3"
)

const (
	KindEc2Topology = "Ec2"

	DefaultVersion    = "datapunch.org/v1alpha1"
	DefaultRegion     = "us-west-1"
	DefaultNamePrefix = "my"
)

type Ec2Topology struct {
	ApiVersion string                     `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                     `json:"kind" yaml:"kind"`
	Metadata   framework.TopologyMetadata `json:"metadata"`
	Spec       Ec2TopologySpec            `json:"spec"`
}

type Ec2TopologySpec struct {
	NamePrefix   string           `json:"namePrefix" yaml:"namePrefix"`
	Region       string           `json:"region" yaml:"region"`
	ImageId      string           `json:"imageId" yaml:"imageId"`
	InstanceType string           `json:"instanceType" yaml:"instanceType"`
	MinCount     int64            `json:"minCount" yaml:"minCount"`
	MaxCount     int64            `json:"maxCount" yaml:"maxCount"`
	InstanceRole resource.IAMRole `json:"instanceRole" yaml:"instanceRole"`
}

func CreateDefaultEc2Topology(namePrefix string) Ec2Topology {
	topologyName := fmt.Sprintf("%s-ec2", namePrefix)
	instanceRoleName := fmt.Sprintf("%s-ec2-instance", namePrefix)
	topology := Ec2Topology{
		ApiVersion: DefaultVersion,
		Kind:       KindEc2Topology,
		Metadata: framework.TopologyMetadata{
			Name:               topologyName,
			CommandEnvironment: map[string]string{},
			Notes:              map[string]string{},
		},
		Spec: Ec2TopologySpec{
			NamePrefix:   namePrefix,
			Region:       DefaultRegion,
			ImageId:      "ami-03af6a70ccd8cb578",
			InstanceType: "t2.micro",
			MinCount:     1,
			MaxCount:     1,
			InstanceRole: resource.IAMRole{
				Name:                     instanceRoleName,
				AssumeRolePolicyDocument: framework.DefaultEC2AssumeRolePolicyDocument,
				ExtraPolicyArns: []string{
					"arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore",
				},
			},
		},
	}
	return topology
}

func (t *Ec2Topology) GetKind() string {
	return t.Kind
}

func (t *Ec2Topology) ToString() string {
	topologyBytes, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology: %s)", err.Error())
	}
	return string(topologyBytes)
}
