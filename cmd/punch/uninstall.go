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
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/hivemetastore"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/kafkaonmsk"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/rdsdatabase"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/sparkoneks"
	"github.com/spf13/cobra"
	"log"
)

var deleteCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall topology",
	Long:  `Delete resources in the topology.`,
	Run: func(cmd *cobra.Command, args []string) {
		topology := getTopologyFromArguments(args)

		kind := topology.GetKind()
		handler := getTopologyHandlerOrFatal(kind)

		resolvedTopology, err := handler.Validate(topology, framework.PhaseBeforeUninstall)
		if err != nil {
			log.Fatalf("Failed to resolve topology: %s", err.Error())
		}

		if DryRun {
			log.Println("Dry run, exit now")
			return
		}

		log.Printf("Uninstalling topology...")

		deploymentOutput, err := handler.Uninstall(resolvedTopology)
		if err != nil {
			log.Fatalf("Failed to delete topology: %s", err.Error())
		}
		if deploymentOutput == nil {
			log.Fatalf("Install failed due to null deployment output")
		}
		log.Printf("Uninstall finished")

		jsonFormat := false
		deploymentOutputStr := framework.MarshalDeploymentOutput(topology.GetKind(), deploymentOutput, jsonFormat)
		log.Printf("----- Uninstall Output -----\n%s", deploymentOutputStr)
	},
}

func init() {
	AddFileNameCommandFlag(deleteCmd)
	AddKeyValueCommandFlags(deleteCmd)
	AddDryRunCommandFlag(deleteCmd)
}
