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

package kafkawithbridge

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/kafkaonmsk"
	"gopkg.in/yaml.v3"
	"regexp"
)

var nonAlphanumericRegexp *regexp.Regexp

func init() {
	framework.DefaultTopologyHandlerManager.AddHandler(KindKafkaTopology, &TopologyHandler{})

	var err error
	nonAlphanumericRegexp, err = regexp.Compile("[^a-zA-Z]+")
	if err != nil {
		panic(err)
	}
}

type TopologyHandler struct {
}

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	topology := GenerateDefaultTopology()
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	topology := GenerateDefaultTopology()
	err := yaml.Unmarshal(yamlContent, &topology)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML (%s): \n%s", err.Error(), string(yamlContent))
	}
	return &topology, nil
}

func (t *TopologyHandler) Validate(topology framework.Topology, phase string) (framework.Topology, error) {
	return topology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	kafkaTopology := topology.(*KafkaTopology)
	deployment := kafkaonmsk.CreateInstallDeployment(kafkaTopology.Spec.KafkaOnMskSpec)
	err := deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	kafkaTopology := topology.(*KafkaTopology)
	deployment := kafkaonmsk.CreateUninstallDeployment(kafkaTopology.Spec.KafkaOnMskSpec)
	err := deployment.Run()
	return deployment.GetOutput(), err
}

