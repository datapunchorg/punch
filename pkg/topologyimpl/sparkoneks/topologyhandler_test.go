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

package sparkoneks

import (
	"log"
	"testing"

	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/stretchr/testify/assert"
)

func TestGenerateTopology(t *testing.T) {
	handler := &TopologyHandler{}
	topology, err := handler.Generate()
	assert.Equal(t, nil, err)
	assert.Equal(t, "SparkOnEks", topology.GetKind())

	log.Printf("-----\n%s\n-----\n", framework.TopologyString(topology))

	sparkTopology := topology.(*SparkOnEksTopology)
	assert.Equal(t, "{{ or .Values.namePrefix `my` }}", sparkTopology.Spec.Eks.NamePrefix)
}

func TestParseTopology(t *testing.T) {
	handler := &TopologyHandler{}
	topology, err := handler.Generate()
	sparkTopology := topology.(*SparkOnEksTopology)
	sparkTopology.Spec.Eks.NamePrefix = "foo"
	yamlContent := framework.TopologyString(topology)

	topology, err = handler.Parse([]byte(yamlContent))
	assert.Equal(t, nil, err)
	assert.Equal(t, "SparkOnEks", topology.GetKind())
	sparkTopology = topology.(*SparkOnEksTopology)
	assert.Equal(t, "foo", sparkTopology.Spec.Eks.NamePrefix)
}

func TestResolveTopology(t *testing.T) {
	handler := &TopologyHandler{}
	topology, err := handler.Generate()
	env := map[string]string{
		"nginxHelmChart":         "../../../third-party/helm-charts/ingress-nginx/charts/ingress-nginx",
		"sparkOperatorHelmChart": "../../../third-party/helm-charts/spark-operator-service/charts/spark-operator-chart",
	}
	topology.(*SparkOnEksTopology).Metadata.CommandEnvironment = env
	topology.(*SparkOnEksTopology).Spec.Spark.Gateway.Password = "aaa"
	resolvedTopology, err := handler.Validate(topology, framework.PhaseBeforeInstall)
	assert.Equal(t, nil, err)
	assert.Equal(t, "SparkOnEks", resolvedTopology.GetKind())
}

func TestResolveTopology_NoPassword(t *testing.T) {
	handler := &TopologyHandler{}
	topology, err := handler.Generate()
	env := map[string]string{
		"nginxHelmChart":         "../../../third-party/helm-charts/ingress-nginx/charts/ingress-nginx",
		"sparkOperatorHelmChart": "../../../third-party/helm-charts/spark-operator-service/charts/spark-operator-chart",
	}
	topology.(*SparkOnEksTopology).Metadata.CommandEnvironment = env
	topology.(*SparkOnEksTopology).Spec.Spark.Gateway.Password = ""
	resolvedTopology, err := handler.Validate(topology, framework.PhaseBeforeInstall)
	assert.NotEqual(t, nil, err)
	assert.Equal(t, nil, resolvedTopology)
}
