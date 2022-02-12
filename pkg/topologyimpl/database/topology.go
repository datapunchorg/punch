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

package database

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/resource"
	"gopkg.in/yaml.v3"
)

const (
	DefaultUserName = "user1"

	KindDatabaseTopology = "Database"

	FieldMaskValue = "***"

	DefaultVersion    = "datapunch.org/v1alpha1"
	DefaultRegion     = "us-west-1"
	DefaultNamePrefix = "my"
)

type DatabaseTopology struct {
	ApiVersion string                     `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                     `json:"kind" yaml:"kind"`
	Metadata   framework.TopologyMetadata `json:"metadata"`
	Spec       DatabaseTopologySpec       `json:"spec"`
}

type DatabaseTopologySpec struct {
	NamePrefix        string   `json:"namePrefix" yaml:"namePrefix"`
	Region            string   `json:"region" yaml:"region"`
	VpcId             string   `json:"vpcId" yaml:"vpcId"`
	AvailabilityZones []string `json:"availabilityZones" yaml:"availabilityZones"`
	DatabaseId        string   `json:"databaseId" yaml:"databaseId"`
	MasterUserName    string   `json:"masterUserName" yaml:"masterUserName"`
	// password must not shorter than 8 characters
	MasterUserPassword string                   `json:"masterUserPassword" yaml:"masterUserPassword"`
	SecurityGroups     []resource.SecurityGroup `json:"securityGroups" yaml:"securityGroups"`
}

func CreateDefaultDatabaseTopology(namePrefix string) DatabaseTopology {
	topologyName := fmt.Sprintf("%s-db", namePrefix)
	securityGroupName := fmt.Sprintf("%s-sg-01", namePrefix)
	topology := DatabaseTopology{
		ApiVersion: DefaultVersion,
		Kind:       KindDatabaseTopology,
		Metadata: framework.TopologyMetadata{
			Name:               topologyName,
			CommandEnvironment: map[string]string{},
			Notes:              map[string]string{},
		},
		Spec: DatabaseTopologySpec{
			NamePrefix:         namePrefix,
			Region:             DefaultRegion,
			VpcId:              "{{ or .Values.vpcId .DefaultVpcId }}",
			AvailabilityZones:  []string{"us-west-1a"},
			DatabaseId:         fmt.Sprintf("%s-db", namePrefix),
			MasterUserName:     DefaultUserName,
			MasterUserPassword: "{{ .Values.masterUserPassword }}",
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

func (t *DatabaseTopology) GetKind() string {
	return t.Kind
}

func (t *DatabaseTopology) ToString() string {
	topologyBytes, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology: %s)", err.Error())
	}
	var copy DatabaseTopology
	err = yaml.Unmarshal(topologyBytes, &copy)
	if err != nil {
		return fmt.Sprintf("(Failed to deserialize topology in ToYamlString(): %s)", err.Error())
	}
	if copy.Spec.MasterUserPassword != "" {
		copy.Spec.MasterUserPassword = FieldMaskValue
	}
	topologyBytes, err = yaml.Marshal(copy)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology in ToString(): %s)", err.Error())
	}
	return string(topologyBytes)
}
