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
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/database"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/kafka"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/sparkonk8s"
	"github.com/spf13/cobra"
	"log"
)

var provisionCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a topology",
	Long:  `Provision resources in the topology.`,
	Run: func(cmd *cobra.Command, args []string) {
		topology := getTopologyFromArguments(args)
		log.Printf("----- Input Topology -----\n%s", MarshalTopology(topology))

		kind := topology.GetKind()
		handler := getTopologyHandlerOrFatal(kind)

		commandEnvironment := createKeyValueMap(CommandEnv)
		templateValues := createKeyValueMap(TemplateValues)
		templateData := framework.CreateTemplateData(commandEnvironment, templateValues)
		resolvedTopology, err := handler.Resolve(topology, &templateData)
		if err != nil {
			log.Fatalf("Failed to resolve topology: %s", err.Error())
		}

		log.Printf("----- Resolved Topology -----\n%s", resolvedTopology.ToString())

		if DryRun {
			log.Println("Dry run, exit now")
			return
		}

		log.Printf("Installing topology...")

		deploymentOutput, err := handler.Install(resolvedTopology)
		if err != nil {
			log.Fatalf("Failed to provision topology: %s", err.Error())
		}

		log.Printf("Install finished")
		if deploymentOutput != nil {
			deploymentOutput := MarshalDeploymentOutput(deploymentOutput)
			log.Printf("----- Install Output -----\n%s", deploymentOutput)
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
	AddPrintUsageExampleCommandFlags(provisionCmd)
}
