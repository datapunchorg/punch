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

package kubelib

import (
	"fmt"
	"log"
	"os/exec"
)

func CheckKubectl(exeLocation string) error {
	cmd := exec.Command(exeLocation, "version", "--client")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("%s not installed: %s", exeLocation, err.Error())
	}
	log.Printf("%s version: %s", exeLocation, string(output))
	return nil
}

func runKubectl(exeLocation string, arguments []string) error {
	cmd := exec.Command(exeLocation, arguments...)
	log.Printf("Running kubectl: %s", cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run %s: %s\n%s", exeLocation, err.Error(), string(output))
	} else {
		log.Printf("finished running %s\n%s", exeLocation, string(output))
		return nil
	}
}

func RunKubectlWithKubeConfig(exeLocation string, kubeConfig KubeConfig, extraArguments []string) error {
	arguments := make([]string, 0, 100)
	arguments = AppendKubectlKubeArguments(arguments, kubeConfig)
	arguments = append(arguments, extraArguments...)
	return runKubectl(exeLocation, arguments)
}

func AppendKubectlKubeArguments(arguments []string, kubeConfig KubeConfig) []string {
	if kubeConfig.ConfigFile != "" {
		arguments = append(arguments, fmt.Sprintf("--kubeconfig=%s", kubeConfig.ConfigFile))
	}
	if kubeConfig.ApiServer != "" {
		arguments = append(arguments, fmt.Sprintf("--server=%s", kubeConfig.ApiServer))
	}
	if kubeConfig.CAFile != "" {
		arguments = append(arguments, fmt.Sprintf("--certificate-authority=%s", kubeConfig.CAFile))
	}
	if kubeConfig.KubeToken != "" {
		arguments = append(arguments, fmt.Sprintf("--token=%s", kubeConfig.KubeToken))
	}
	return arguments
}
