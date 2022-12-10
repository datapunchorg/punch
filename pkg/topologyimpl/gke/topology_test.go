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

package eks

import (
	"bytes"
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
	"text/template"
)

func TestTemplate(t *testing.T) {
	d := framework.CreateTemplateData(nil)

	topology := CreateDefaultEksTopology("my", "{{ or .Values.s3BucketName .DefaultS3BucketName }}")
	topology.Metadata.CommandEnvironment["kubeConfig"] = ""
	topology.Metadata.CommandEnvironment["helmExecutable"] = "helm"
	topology.Spec.EksCluster.ClusterName = "{{ .Values.eksCluster.name }}"

	tmpl, err := template.New("").Parse(framework.TopologyString(&topology))
	assert.Equal(t, nil, err)

	data := framework.CreateTemplateDataWithRegion(&d)
	data.AddValue("s3BucketName", "bucket123abc")
	data.AddValue("eksCluster", map[string]interface{}{"name": "cluster1"})

	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, &data)
	assert.Equal(t, nil, err)

	str := buffer.String()
	fmt.Println(str)

	eksTopology := EksTopology{}
	yaml.Unmarshal([]byte(str), &eksTopology)
	assert.Equal(t, "bucket123abc", eksTopology.Spec.S3BucketName)
	assert.Equal(t, "cluster1", eksTopology.Spec.EksCluster.ClusterName)
}

func TestTemplateWithAlternativeValue(t *testing.T) {
	d := framework.CreateTemplateData(nil)

	topology := CreateDefaultEksTopology("my", "{{ or .Values.s3BucketName `abcde12345` }}")
	topology.Metadata.CommandEnvironment["kubeConfig"] = ""
	topology.Metadata.CommandEnvironment["helmExecutable"] = "helm"

	tmpl, err := template.New("").Parse(framework.TopologyString(&topology))
	assert.Equal(t, nil, err)

	data := framework.CreateTemplateDataWithRegion(&d)

	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, &data)
	assert.Equal(t, nil, err)

	str := buffer.String()
	fmt.Println(str)

	eksTopology := EksTopology{}
	yaml.Unmarshal([]byte(str), &eksTopology)
	assert.Equal(t, "abcde12345", eksTopology.Spec.S3BucketName)
	assert.Equal(t, "", eksTopology.Metadata.CommandEnvironment["kubeConfig"])
	assert.Equal(t, "helm", eksTopology.Metadata.CommandEnvironment["helmExecutable"])
}

func TestTemplateWithUnresolvedValue(t *testing.T) {
	d := framework.CreateTemplateData(nil)

	topology := CreateDefaultEksTopology("my", "{{ .Values.s3BucketName }}")
	tmpl, err := template.New("").Parse(framework.TopologyString(&topology))
	assert.Equal(t, nil, err)

	data := framework.CreateTemplateDataWithRegion(&d)

	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, &data)
	assert.Equal(t, nil, err)

	str := buffer.String()
	fmt.Println(str)

	eksTopology := EksTopology{}
	yaml.Unmarshal([]byte(str), &eksTopology)
	assert.Equal(t, "<no value>", eksTopology.Spec.S3BucketName)
}
