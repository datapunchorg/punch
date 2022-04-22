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

package main

import (
	"bytes"
	"github.com/datapunchorg/punch/pkg/framework"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"strings"
	"text/template"
)

var Output string

var FileName string

var CommandEnv []string
var TemplateValues []string
var PatchValues []string

var DryRun bool

var PrintUsageExample bool

func createKeyValueMap(keyValuePairs []string) map[string]string {
	result := map[string]string{}
	for _, entry := range keyValuePairs {
		index := strings.Index(entry, "=")
		if index == -1 {
			result[entry] = ""
		} else {
			key := entry[0:index]
			value := entry[index+1:]
			result[key] = value
		}
	}
	return result
}

func getTopologyFromArguments(args []string) framework.Topology {
	fileName := FileName
	var inputTopology []byte
	if fileName == "" {
		if len(args) == 0 {
			log.Fatalf("Please specify an argument to identify the kind of topology, or specify -f to provide a topology file")
		}
		kind := args[0]
		handler := getTopologyHandlerOrFatal(kind)
		generatedTopology, err := handler.Generate()
		if err != nil {
			log.Fatalf("Failed to generate topology: %s", err.Error())
		}
		topologyBytes, err := yaml.Marshal(generatedTopology)
		if err != nil {
			log.Fatalf("Failed to marshal topology: %s", err.Error())
		}
		inputTopology = topologyBytes
	} else {
		fileContent, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Fatalf("Failed to read topology file %s: %s", fileName, err.Error())
		}
		inputTopology = fileContent
	}
	transformedTopology := transformTopologyTemplate(string(inputTopology))
	transformedTopologyBytes := []byte(transformedTopology)
	kind := getKind(transformedTopologyBytes)
	handler := getTopologyHandlerOrFatal(kind)
	topology, err := handler.Parse(transformedTopologyBytes)
	if err != nil {
		log.Fatalf("Failed to parse topology file %s: %s", fileName, err.Error())
	}
	return topology
}

type structWithKind struct {
	Kind string `json:"kind" yaml:"kind"`
}

func getKind(yamlContent []byte) string {
	s := structWithKind{}
	yaml.Unmarshal(yamlContent, &s)
	return s.Kind
}

func transformTopologyTemplate(content string) string {
	tmpl, err := template.New("").Parse(content) // .Option("missingkey=error")?
	if err != nil {
		log.Fatalf("Failed to parse topology template, error: %s, template: %s", err.Error(), content)
	}

	commandEnvironment := createKeyValueMap(CommandEnv)
	templateValues := createKeyValueMap(TemplateValues)
	templateData := framework.CreateTemplateData(commandEnvironment, templateValues)
	templateDataWithRegion := framework.CreateTemplateDataWithRegion(&templateData)

	buffer := bytes.Buffer{}
	err = tmpl.Execute(&buffer, &templateDataWithRegion)
	if err != nil {
		log.Fatalf("Failed to execute topology template: %s", err.Error())
	}
	transformedContent := buffer.String()
	return transformedContent
}
