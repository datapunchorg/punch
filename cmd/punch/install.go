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
	"log"
	"strings"

	"github.com/datapunchorg/punch/pkg/framework"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/kafkabridge"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/kafkaonmsk"
	"github.com/spf13/cobra"
)

var provisionCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a topology",
	Long:  `Provision resources in the topology.`,
	Run: func(cmd *cobra.Command, args []string) {
		topology := getTopologyFromArguments(args)

		kind := topology.GetKind()
		handler := getTopologyHandlerPlugin(kind)

		resolvedTopology, err := handler.Validate(topology, framework.PhaseBeforeInstall)
		if err != nil {
			log.Fatalf("Failed to resolve topology: %s", err.Error())
		}

		if DryRun {
			log.Println("Dry run, exit now")
			return
		}

		log.Printf("Installing topology...")

		deploymentOutput, err := handler.Install(resolvedTopology)
		if err != nil {
			log.Fatalf("Failed to provision topology: %s", err.Error())
		}
		if deploymentOutput == nil {
			log.Fatalf("Install failed due to null deployment output")
		}
		log.Printf("Install finished")

		outputFile := Output
		jsonFormat := strings.HasSuffix(strings.ToLower(outputFile), ".json")
		deploymentOutputStr := framework.MarshalDeploymentOutput(topology.GetKind(), deploymentOutput, jsonFormat)
		if outputFile == "" {
			log.Printf("----- Install Output -----\n%s", deploymentOutputStr)
		} else {
			writeOutputFile(outputFile, deploymentOutputStr)
		}

		if PrintUsageExample {
			ableToPrintUsageExample, ok := handler.(framework.AbleToPrintUsageExample)
			if ok {
				ableToPrintUsageExample.PrintUsageExample(resolvedTopology, deploymentOutput)
			} else {
				log.Printf("Topology handler does not support printing usage example")
			}
		}
	},
}

func init() {
	AddFileNameCommandFlag(provisionCmd)
	AddKeyValueCommandFlags(provisionCmd)
	AddDryRunCommandFlag(provisionCmd)
	AddOutputCommandFlag(provisionCmd)
	AddPrintUsageExampleCommandFlags(provisionCmd)
}
