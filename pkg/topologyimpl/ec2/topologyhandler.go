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
	"bytes"
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"gopkg.in/yaml.v3"
	"log"
	"text/template"
)

const MaxInstanceCount = 1000

func init() {
	framework.DefaultTopologyHandlerManager.AddHandler(KindEc2Topology, &TopologyHandler{})
}

type TopologyHandler struct {
}

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	namePrefix := "{{ or .Values.namePrefix `my` }}"
	topology := CreateDefaultEc2Topology(namePrefix)
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	result := CreateDefaultEc2Topology(DefaultNamePrefix)
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

	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to execute topology template: %s", err.Error())
	}
	resolvedContent := buffer.String()
	resolvedTopology, err := t.Parse([]byte(resolvedContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse resolved topology (%s): %s", err.Error(), resolvedContent)
	}

	resolvedEc2Topology := resolvedTopology.(*Ec2Topology)
	if resolvedEc2Topology.Spec.MinCount < 1 {
		return nil, fmt.Errorf("Spec.MinCount has invalid value: %d", resolvedEc2Topology.Spec.MinCount)
	}
	if resolvedEc2Topology.Spec.MaxCount < 1 {
		return nil, fmt.Errorf("Spec.MaxCount has invalid value: %d", resolvedEc2Topology.Spec.MaxCount)
	}
	if resolvedEc2Topology.Spec.MaxCount > MaxInstanceCount {
		return nil, fmt.Errorf("Spec.MaxCount is too large: %d", resolvedEc2Topology.Spec.MaxCount)
	}

	return resolvedTopology, nil
}

// TODO check instance status after install/uninstall

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	deployment := framework.NewDeployment()
	deployment.AddStep("createInstances", "Create instances", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
		ec2Topology := t.(*Ec2Topology)
		// TODO check existing running instances and only create new instances when needed
		result, err := CreateInstances(ec2Topology.Metadata.Name, ec2Topology.Spec)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		var instanceIds []string
		for _, entry := range result {
			instanceIds = append(instanceIds, *entry.InstanceId)
		}
		return framework.DeploymentStepOutput{"instanceIds": instanceIds}, nil
	})
	err := deployment.RunSteps(topology)
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	deployment := framework.NewDeployment()
	deployment.AddStep("deleteInstances", "Delete instances", func(c framework.DeploymentContext, t framework.Topology) (framework.DeploymentStepOutput, error) {
		ec2Topology := t.(*Ec2Topology)
		instanceIds, err := GetInstanceIdsByTopology(ec2Topology.Spec.Region, ec2Topology.Metadata.Name)
		if err != nil {
			return nil, err
		}
		if len(instanceIds) == 0 {
			log.Printf("Found no instances for topology %s, do not delete any instances", ec2Topology.Metadata.Name)
			return framework.NewDeploymentStepOutput(), nil
		}
		err = DeleteInstances(ec2Topology.Spec.Region, instanceIds)
		if err != nil {
			return nil, err
		}
		return framework.NewDeploymentStepOutput(), nil
	})
	err := deployment.RunSteps(topology)
	return deployment.GetOutput(), err
}
