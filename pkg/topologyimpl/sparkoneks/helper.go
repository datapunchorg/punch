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

package sparkoneks

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"strings"
)

func ValidateSparkComponentSpec(sparkComponentSpec SparkComponentSpec, metadata framework.TopologyMetadata, phase string) error {
	if strings.EqualFold(phase, framework.PhaseBeforeInstall) {
		if sparkComponentSpec.Gateway.Password == "" || sparkComponentSpec.Gateway.Password == framework.TemplateNoValue {
			return fmt.Errorf("spec.spark.gateway.password is emmpty, please provide the value for the password")
		}
	}
	err := framework.CheckCmdEnvFolderExists(metadata, CmdEnvSparkOperatorHelmChart)
	if err != nil {
		return err
	}
	return nil
}
