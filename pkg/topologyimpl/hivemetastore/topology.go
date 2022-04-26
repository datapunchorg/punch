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

package hivemetastore

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"gopkg.in/yaml.v3"
)

const (
	KindHiveMetastoreTopology = "HiveMetastore"

	FieldMaskValue = "***"

	DefaultVersion    = "datapunch.org/v1alpha1"
	DefaultRegion     = "us-west-1"
	DefaultNamePrefix = "my"
)

type HiveMetastoreTopology struct {
	framework.TopologyBase               `json:",inline" yaml:",inline"`
	Spec       HiveMetastoreTopologySpec      `json:"spec"`
}

type HiveMetastoreTopologySpec struct {
	NamePrefix        string   `json:"namePrefix" yaml:"namePrefix"`
	Region            string   `json:"region" yaml:"region"`
	DbConnectionString             string   `json:"dbConnectionString" yaml:"dbConnectionString"`
	DbUserName string `json:"dbUserName" yaml:"dbUserName"`
	DbUserPassword string `json:"dbUserPassword" yaml:"dbUserPassword"`
}

func CreateDefaultHiveMetastoreTopology(namePrefix string) HiveMetastoreTopology {
	topologyName := fmt.Sprintf("%s-db-01", namePrefix)
	topology := HiveMetastoreTopology{
		TopologyBase: framework.TopologyBase{
			ApiVersion: DefaultVersion,
			Kind: KindHiveMetastoreTopology,
			Metadata: framework.TopologyMetadata{
				Name:               topologyName,
				CommandEnvironment: map[string]string{},
				Notes:              map[string]string{},
			},
		},
		Spec: HiveMetastoreTopologySpec{
			NamePrefix:         namePrefix,
			Region:             DefaultRegion,
			DbConnectionString: "{{ or .Values.dbConnectionString 'TODO_REQUIRED_FIELD' }}",
			DbUserName:      "{{ or .Values.dbUserName 'TODO_REQUIRED_FIELD' }}",
			DbUserPassword: "{{ or .Values.dbUserPassword 'TODO_REQUIRED_FIELD' }}",
		},
	}
	return topology
}

func (t *HiveMetastoreTopology) GetKind() string {
	return t.Kind
}

func (t *HiveMetastoreTopology) GetSpec() framework.TopologySpecPointer {
	return &t.Spec
}

func (t *HiveMetastoreTopology) String() string {
	topologyBytes, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology: %s)", err.Error())
	}
	var copy HiveMetastoreTopology
	err = yaml.Unmarshal(topologyBytes, &copy)
	if err != nil {
		return fmt.Sprintf("(Failed to deserialize topology in ToYamlString(): %s)", err.Error())
	}
	if copy.Spec.DbUserPassword != "" {
		copy.Spec.DbUserPassword = FieldMaskValue
	}
	topologyBytes, err = yaml.Marshal(copy)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology in String(): %s)", err.Error())
	}
	return string(topologyBytes)
}
