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
	"fmt"
	"io/ioutil"
	"log"
	"os"

	_ "github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/gke"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/hivemetastore"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/kafkabridge"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/kafkaonmsk"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/kyuubioneks"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/pinotquickstart"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/sparkoneks"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/sparkongke"
	_ "github.com/datapunchorg/punch/pkg/topologyimpl/superset"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a topology",
	Long:  `Generate a topology yaml representation.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.SetOutput(ioutil.Discard)

		if len(args) == 0 {
			exitWithError("Please specify an argument to identify which kind of topology to generate")
		}

		kind := args[0]
		handler := getTopologyHandlerPlugin(kind)
		topology, err := handler.Generate()
		if err != nil {
			exitWithError(err.Error())
		}

		generatedContent := MarshalTopology(topology)

		outputFile := Output
		if outputFile == "" {
			fmt.Printf("%s\n", generatedContent)
		} else {
			writeOutputFile(outputFile, generatedContent)
		}
	},
}

func init() {
	AddOutputCommandFlag(generateCmd)
}

func exitWithError(str string) {
	fmt.Fprintln(os.Stderr, str)
	os.Exit(1)
}
