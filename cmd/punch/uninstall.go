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

package main

import (
	"log"

	"github.com/datapunchorg/punch/pkg/framework"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/argocdongke"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/gke"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/hivemetastore"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/kafkabridge"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/kafkaonmsk"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/kyuubioneks"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/pinotquickstart"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/sparkoneks"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/superset"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall topology",
	Long:  `Delete resources in the topology.`,
	Run: func(cmd *cobra.Command, args []string) {
		topology := getTopologyFromArguments(args)

		kind := topology.GetKind()
		handler := getTopologyHandlerPlugin(kind)

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
