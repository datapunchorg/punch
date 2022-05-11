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
	"gopkg.in/yaml.v3"
	"log"
	"strconv"
)

type CommandEnvironment struct {
	valueMap map[string]string
}

func CreateCommandEnvironment(valueMap map[string]string) CommandEnvironment {
	env := CommandEnvironment{
		valueMap: map[string]string{},
	}
	for key, value := range valueMap {
		env.valueMap[key] = value
	}
	return env
}

func (t *CommandEnvironment) Set(key string, value string) {
	t.valueMap[key] = value
}

func (t *CommandEnvironment) Get(key string) string {
	return t.valueMap[key]
}

func (t *CommandEnvironment) GetOrElse(key string, defaultValue string) string {
	return GetOrElse(t.valueMap, key, defaultValue)
}

func (t *CommandEnvironment) GetBoolOrElse(key string, defaultValue bool) bool {
	if value, ok := t.valueMap[key]; ok {
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			log.Printf("Got invalid bool value %s from command environment, will use provided default value: %t", value, defaultValue)
			return defaultValue
		}
		return boolValue
	} else {
		return defaultValue
	}
}

func (t *CommandEnvironment) GetValueMap() map[string]string {
	result := map[string]string{}
	for key, value := range t.valueMap {
		result[key] = value
	}
	return result
}

func (t *CommandEnvironment) ToYamlString() string {
	topologyBytes, err := yaml.Marshal(t.valueMap)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize command environment in ToYamlString(): %s)", err.Error())
	}
	return string(topologyBytes)
}
