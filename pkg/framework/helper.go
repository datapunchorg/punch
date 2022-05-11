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

package framework

import (
	"fmt"
	"os"
)

const FieldMaskValue = "***"

func CheckCmdEnvFolderExists(metadata TopologyMetadata, cmdEnvKey string) error {
	cmdEnvValue := metadata.CommandEnvironment[cmdEnvKey]
	if cmdEnvValue == "" {
		return fmt.Errorf("Metadata.CommandEnvironment[\"%s\"] is empty", cmdEnvKey)
	}
	if _, err := os.Stat(cmdEnvValue); os.IsNotExist(err) {
		return fmt.Errorf("folder not exists (specified in Metadata.CommandEnvironment[\"%s\"]=\"%s\")", cmdEnvKey, cmdEnvValue)
	}
	return nil
}

func GetOrElse(valueMap map[string]string, key string, defaultValue string) string {
	if value, ok := valueMap[key]; ok {
		return value
	} else {
		return defaultValue
	}
}

func CopyMissingKeyValuesFromStringMap(target map[string]string, source map[string]string )  {
	for k, v := range source {
		if _, ok := target[k]; !ok {
			target[k] = v
		}
	}
}
