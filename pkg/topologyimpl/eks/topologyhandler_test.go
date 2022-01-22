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

package eks

import (
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func TestGenerateTopology(t *testing.T) {
	handler := &TopologyHandler{}
	topology, err := handler.Generate()
	assert.Equal(t, nil, err)
	assert.Equal(t, "Eks", topology.GetKind())

	log.Printf("-----\n%s\n-----\n", topology.ToString())

	sparkTopology := topology.(*EksTopology)
	assert.Equal(t, "{{ or .Values.namePrefix `my` }}", sparkTopology.Spec.NamePrefix)
}

func TestParseTopology(t *testing.T) {
	handler := &TopologyHandler{}
	topology, err := handler.Generate()
	sparkTopology := topology.(*EksTopology)
	sparkTopology.Spec.NamePrefix = "foo"
	yamlContent := topology.ToString()

	topology, err = handler.Parse([]byte(yamlContent))
	assert.Equal(t, nil, err)
	assert.Equal(t, "Eks", topology.GetKind())
	sparkTopology = topology.(*EksTopology)
	assert.Equal(t, "foo", sparkTopology.Spec.NamePrefix)
}

func TestResolveTopology(t *testing.T) {
	handler := &TopologyHandler{}
	topology, err := handler.Generate()
	env := map[string]string{
		"nginxHelmChart": "../../../third-party/helm-charts/ingress-nginx/charts/ingress-nginx",
	}
	values := map[string]string{}
	templateData := framework.CreateTemplateData(env, values)
	resolvedTopology, err := handler.Resolve(topology, &templateData)
	assert.Equal(t, nil, err)
	assert.Equal(t, "Eks", resolvedTopology.GetKind())
}

