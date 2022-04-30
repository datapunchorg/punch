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
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

// MarshalTopology gets a YAML string representation for the topology.
// Please use this method with caution, since it will not mask passwords in the topology.
func MarshalTopology(topology framework.Topology) string {
	topologyBytes, err := yaml.Marshal(topology)
	if err != nil {
		return fmt.Sprintf("<Failed to serialize topology: %s>", err.Error())
	}
	return string(topologyBytes)
}

func writeOutputFile(filePath string, fileContent string) {
	f, err := os.Create(filePath)
	if err != nil {
		exitWithError(fmt.Sprintf("Failed to create output file %s: %s", filePath, err.Error()))
	}
	defer f.Close()
	_, err = f.WriteString(fileContent)
	if err != nil {
		exitWithError(fmt.Sprintf("Failed to write output file %s: %s", filePath, err.Error()))
	}
	log.Printf("Wrote output to file %s", filePath)
}
