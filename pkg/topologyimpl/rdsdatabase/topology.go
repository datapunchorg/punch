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

package main

import (
	"fmt"

	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/resource"
	"gopkg.in/yaml.v3"
)

const (
	DefaultUserName = "user1"

	KindDatabaseTopology = "RdsDatabase"
)

type RdsDatabaseTopology struct {
	framework.TopologyBase `json:",inline" yaml:",inline"`
	Spec                   RdsDatabaseTopologySpec `json:"spec"`
}

type RdsDatabaseTopologySpec struct {
	NamePrefix        string   `json:"namePrefix" yaml:"namePrefix"`
	Region            string   `json:"region" yaml:"region"`
	VpcId             string   `json:"vpcId" yaml:"vpcId"`
	AvailabilityZones []string `json:"availabilityZones" yaml:"availabilityZones"`
	DatabaseId        string   `json:"databaseId" yaml:"databaseId"`
	Engine            string   `json:"engine" yaml:"engine"`
	EngineVersion     string   `json:"engineVersion" yaml:"engineVersion"`
	// The DB engine mode of the DB cluster, either provisioned, serverless, parallelquery, global, or multimaster.
	EngineMode     string `json:"engineMode" yaml:"engineMode"`
	MasterUserName string `json:"masterUserName" yaml:"masterUserName"`
	// password must not shorter than 8 characters
	MasterUserPassword string                   `json:"masterUserPassword" yaml:"masterUserPassword"`
	Port               int64                    `json:"port" yaml:"port"`
	SecurityGroups     []resource.SecurityGroup `json:"securityGroups" yaml:"securityGroups"`
}

func CreateDefaultRdsDatabaseTopology(namePrefix string) RdsDatabaseTopology {
	topologyName := fmt.Sprintf("%s-db-01", namePrefix)
	securityGroupName := fmt.Sprintf("%s-sg-01", namePrefix)
	topology := RdsDatabaseTopology{
		TopologyBase: framework.TopologyBase{
			ApiVersion: framework.DefaultVersion,
			Kind:       KindDatabaseTopology,
			Metadata: framework.TopologyMetadata{
				Name:               topologyName,
				CommandEnvironment: map[string]string{},
				Notes:              map[string]string{},
			},
		},
		Spec: RdsDatabaseTopologySpec{
			NamePrefix:         namePrefix,
			Region:             framework.DefaultRegion,
			VpcId:              "{{ or .Values.vpcId .DefaultVpcId }}",
			AvailabilityZones:  []string{"us-west-1a"},
			DatabaseId:         topologyName,
			Engine:             "aurora-postgresql",
			EngineVersion:      "10.14",
			EngineMode:         "serverless",
			MasterUserName:     DefaultUserName,
			MasterUserPassword: "",
			SecurityGroups: []resource.SecurityGroup{
				{
					Name: securityGroupName,
					InboundRules: []resource.SecurityGroupInboundRule{
						{
							IPProtocol: "-1",
							FromPort:   -1,
							ToPort:     -1,
							IPRanges:   []string{"0.0.0.0/0"},
						},
					},
				},
			},
		},
	}
	return topology
}

func (t *RdsDatabaseTopology) GetKind() string {
	return t.Kind
}

func (t *RdsDatabaseTopology) GetMetadata() *framework.TopologyMetadata {
	return &t.Metadata
}

func (t *RdsDatabaseTopology) String() string {
	topologyBytes, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology: %s)", err.Error())
	}
	var copy RdsDatabaseTopology
	err = yaml.Unmarshal(topologyBytes, &copy)
	if err != nil {
		return fmt.Sprintf("(Failed to deserialize topology in ToYamlString(): %s)", err.Error())
	}
	if copy.Spec.MasterUserPassword != "" {
		copy.Spec.MasterUserPassword = framework.FieldMaskValue
	}
	topologyBytes, err = yaml.Marshal(copy)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology in String(): %s)", err.Error())
	}
	return string(topologyBytes)
}
