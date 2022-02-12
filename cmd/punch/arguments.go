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
	"github.com/datapunchorg/punch/pkg/framework"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"strings"
)

var Output string

var FileName string

var CommandEnv []string
var TemplateValues []string

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
	var topology framework.Topology
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
		topology = generatedTopology
	} else {
		fileContent, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Fatalf("Failed to read topology file %s: %s", fileName, err.Error())
		}
		kind := getKind(fileContent)
		handler := getTopologyHandlerOrFatal(kind)
		topology, err = handler.Parse(fileContent)
		if err != nil {
			log.Fatalf("Failed to parse topology file %s: %s", fileName, err.Error())
		}
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
