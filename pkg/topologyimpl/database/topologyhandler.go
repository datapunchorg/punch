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
	"bytes"
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"gopkg.in/yaml.v3"
	"regexp"
	"text/template"
)

var nonAlphanumericRegexp *regexp.Regexp

func init() {
	framework.DefaultTopologyHandlerManager.AddHandler(KindDatabaseTopology, &TopologyHandler{})

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
	topology := CreateDefaultDatabaseTopology(namePrefix)
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	result := CreateDefaultDatabaseTopology(DefaultNamePrefix)
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

	databaseData := CreateDatabaseTemplateData(data)

	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, &databaseData)
	if err != nil {
		return nil, fmt.Errorf("failed to execute topology template: %s", err.Error())
	}
	resolvedContent := buffer.String()
	resolvedTopology, err := t.Parse([]byte(resolvedContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse resolved topology (%s): %s", err.Error(), resolvedContent)
	}

	resolvedDatabaseTopology := resolvedTopology.(*DatabaseTopology)
	if resolvedDatabaseTopology.Spec.MasterUserPassword == "" || resolvedDatabaseTopology.Spec.MasterUserPassword == framework.TemplateNoValue {
		return nil, fmt.Errorf("spec.masterUserPassword is emmpty, please provide the value for the password")
	}

	return resolvedTopology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	deployment := framework.NewDeployment()
	deployment.AddStep("createDatabase", "Create database", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
		databaseTopology := t.(*DatabaseTopology)
		result, err := CreateDatabase(databaseTopology.Spec)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		return framework.DeploymentStepOutput{"endpoint": *result.Endpoint}, nil
	})
	err := deployment.RunSteps(topology)
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	deployment := framework.NewDeployment()
	deployment.AddStep("deleteDatabase", "Delete database", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
		databaseTopology := t.(*DatabaseTopology)
		region := databaseTopology.Spec.Region
		databaseId := databaseTopology.Spec.DatabaseId
		err := DeleteDatabase(region, databaseId)
		return framework.NewDeploymentStepOutput(), err
	})
	err := deployment.RunSteps(topology)
	return deployment.GetOutput(), err
}

// The name for your database of up to 64 alphanumeric characters.
// TODO check length
func normalizeDatabaseName(name string) string {
	return nonAlphanumericRegexp.ReplaceAllString(name, "")
}
