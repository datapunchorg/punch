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

package kubelib

import (
	"log"
	"os/exec"
)

func CheckHelmOrFatal(helmExeLocation string) {
	helmCmd := exec.Command(helmExeLocation, "version")
	helmOut, err := helmCmd.Output()
	if err != nil {
		log.Fatalf("helm not installed: %v", err)
	}
	log.Printf("helm version: %s", string(helmOut))
}

// TODO return error
func InstallHelm(helmExeLocation string, helmChartLocation string, kubeConfig KubeConfig, extraArguments []string, installName string, namespace string) {
	arguments := []string{"install", installName, helmChartLocation,
		"--namespace", namespace, "--create-namespace"}

	arguments = AppendHelmKubeArguments(arguments, kubeConfig)
	arguments = append(arguments, extraArguments...)

	helmCmd := exec.Command(helmExeLocation, arguments...)
	log.Printf("Running helm: %s", helmCmd.String())
	helmOut, err := helmCmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to run helm\n%s\n%s", err.Error(), string(helmOut))
		// TODO check error "INSTALLATION FAILED: cannot re-use a name that is still in use" and ask user to uninstall
	} else {
		log.Printf("Finished running helm\n%s", string(helmOut))
	}
}
