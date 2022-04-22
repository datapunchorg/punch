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
	"github.com/spf13/cobra"
	"log"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "punch",
	Short: "punch is the command-line tool to deploy data analytics platform",
	Long:  "punch is the command-line tool to deploy data analytics platform",
}

func init() {
	rootCmd.AddCommand(provisionCmd, deleteCmd, generateCmd)
}

func AddOutputCommandFlag(command *cobra.Command) {
	command.Flags().StringVarP(&Output, "output", "o", "",
		"output file which contains the generated content")
}

func AddFileNameCommandFlag(command *cobra.Command) {
	command.Flags().StringVarP(&FileName, "filename", "f", "",
		"file which contains the configuration to deploy")
}

func AddKeyValueCommandFlags(command *cobra.Command) {
	command.Flags().StringSliceVarP(&CommandEnv, "env", "", []string{},
		"Command environment, e.g. key1=value1")
	command.Flags().StringSliceVarP(&TemplateValues, "set", "", []string{},
		"Template value, e.g. key1=value1")
	command.Flags().StringSliceVarP(&PatchValues, "patch", "", []string{},
		"Patching value on spec fields, e.g. field1=value1")
}

func AddDryRunCommandFlag(command *cobra.Command) {
	command.Flags().BoolVarP(&DryRun, "dryrun", "", false,
		"Dry run to print out information without really making change")
}

func AddPrintUsageExampleCommandFlags(command *cobra.Command) {
	command.Flags().BoolVarP(&PrintUsageExample, "print-usage-example", "", false,
		"print out usage example (IMPORTANT: maybe insecure since the example may contain user password, please use this with caution)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
	}
}

func getTopologyHandlerOrFatal(kind string) framework.TopologyHandler {
	handler := framework.DefaultTopologyHandlerManager.GetHandler(kind)
	if handler == nil {
		log.Fatalf("Topology kind %s not supported", kind)
	}
	return handler
}
