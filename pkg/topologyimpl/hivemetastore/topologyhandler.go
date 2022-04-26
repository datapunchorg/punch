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
	"regexp"
	"strings"
)

var nonAlphanumericRegexp *regexp.Regexp

func init() {
	framework.DefaultTopologyHandlerManager.AddHandler(KindHiveMetastoreTopology, &TopologyHandler{})

	var err error
	nonAlphanumericRegexp, err = regexp.Compile("[^a-zA-Z]+")
	if err != nil {
		panic(err)
	}
}

type TopologyHandler struct {
}

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	namePrefix := "{{ or .Values.namePrefix `my` }}"
	topology := CreateDefaultHiveMetastoreTopology(namePrefix)
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	result := CreateDefaultHiveMetastoreTopology(DefaultNamePrefix)
	err := yaml.Unmarshal(yamlContent, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML (%s): \n%s", err.Error(), string(yamlContent))
	}
	return &result, nil
}

func (t *TopologyHandler) Validate(topology framework.Topology, phase string) (framework.Topology, error) {
	resolvedDatabaseTopology := topology.(*HiveMetastoreTopology)

	if strings.EqualFold(phase, framework.PhaseBeforeInstall) {
		if resolvedDatabaseTopology.Spec.DbUserPassword == "" || resolvedDatabaseTopology.Spec.DbUserPassword == framework.TemplateNoValue {
			return nil, fmt.Errorf("spec.dbUserPassword is emmpty, please provide the value for the password")
		}
	}

	return topology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	databaseTopology := topology.(*HiveMetastoreTopology)
	deployment := framework.NewDeployment()
	deployment.AddStep("initHiveMetastoreDatabase", "Init Hive Metastore database", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
		return framework.DeploymentStepOutput{"TODO": databaseTopology.Metadata.Name}, nil
	})
	err := deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	deployment := framework.NewDeployment()
	return deployment.GetOutput(), nil
}
