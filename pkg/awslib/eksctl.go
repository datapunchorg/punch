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

package awslib

import (
	"fmt"
	"log"
	"os/exec"
)

func CheckEksCtlCmd(exeLocation string) error {
	cmd := exec.Command(exeLocation, "version")
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("eksctl not installed: %s", err.Error())
	}
	return nil
}

func RunEksCtlCmd(exeLocation string, arguments []string) {
	cmd := exec.Command(exeLocation, arguments...)
	log.Printf("Running eksctl: %s", cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to run eksctl\n%s\n%s", err.Error(), string(out))
	} else {
		log.Printf("Finished running eksctl\n%s", string(out))
	}
}
