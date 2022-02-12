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

package sparkonk8s

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
	d := framework.CreateTemplateData(nil, nil)

	topology := CreateDefaultSparkTopology("my", "{{ or .Values.s3BucketName .DefaultS3BucketName }}")
	topology.Metadata.CommandEnvironment["kubeConfig"] = "{{ or .Env.kubeConfig `` }}"
	topology.Metadata.CommandEnvironment["helmExecutable"] = "{{ or .Env.helmExecutable `helm` }}"
	topology.Spec.EKS.ClusterName = "{{ .Values.eksCluster.name }}"

	tmpl, err := template.New("").Parse(topology.ToString())
	assert.Equal(t, nil, err)

	data := CreateSparkTemplateData(&d)
	data.AddValue("s3BucketName", "bucket123abc")
	data.AddValue("eksCluster", map[string]interface{}{"name": "cluster1"})
	data.AddEnv("kubeConfig", "./foo/kube.config")
	data.AddEnv("helmExecutable", "./bar/helm")

	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, &data)
	assert.Equal(t, nil, err)

	str := buffer.String()
	fmt.Println(str)

	sparkTopology := SparkTopology{}
	yaml.Unmarshal([]byte(str), &sparkTopology)
	assert.Equal(t, "bucket123abc", sparkTopology.Spec.S3BucketName)
	assert.Equal(t, "cluster1", sparkTopology.Spec.EKS.ClusterName)
	assert.Equal(t, "./foo/kube.config", sparkTopology.Metadata.CommandEnvironment["kubeConfig"])
	assert.Equal(t, "./bar/helm", sparkTopology.Metadata.CommandEnvironment["helmExecutable"])
}

func TestTemplateWithAlternativeValue(t *testing.T) {
	d := framework.CreateTemplateData(nil, nil)

	topology := CreateDefaultSparkTopology("my", "{{ or .Values.s3BucketName `abcde12345` }}")
	topology.Metadata.CommandEnvironment["kubeConfig"] = "{{ or .Env.kubeConfig `` }}"
	topology.Metadata.CommandEnvironment["helmExecutable"] = "{{ or .Env.helmExecutable `helm` }}"

	tmpl, err := template.New("").Parse(topology.ToString())
	assert.Equal(t, nil, err)

	data := CreateSparkTemplateData(&d)

	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, &data)
	assert.Equal(t, nil, err)

	str := buffer.String()
	fmt.Println(str)

	sparkTopology := SparkTopology{}
	yaml.Unmarshal([]byte(str), &sparkTopology)
	assert.Equal(t, "abcde12345", sparkTopology.Spec.S3BucketName)
	assert.Equal(t, "", sparkTopology.Metadata.CommandEnvironment["kubeConfig"])
	assert.Equal(t, "helm", sparkTopology.Metadata.CommandEnvironment["helmExecutable"])
}

func TestTemplateWithUnresolvedValue(t *testing.T) {
	d := framework.CreateTemplateData(nil, nil)

	topology := CreateDefaultSparkTopology("my", "{{ .Values.s3BucketName }}")
	tmpl, err := template.New("").Parse(topology.ToString())
	assert.Equal(t, nil, err)

	data := CreateSparkTemplateData(&d)

	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, &data)
	assert.Equal(t, nil, err)

	str := buffer.String()
	fmt.Println(str)

	sparkTopology := SparkTopology{}
	yaml.Unmarshal([]byte(str), &sparkTopology)
	assert.Equal(t, "<no value>", sparkTopology.Spec.S3BucketName)
}
