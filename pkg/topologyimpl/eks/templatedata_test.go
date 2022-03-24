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

package eks

import (
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAddValueWithNestedKey(t *testing.T) {
	d := framework.CreateTemplateData(nil, nil)
	data := CreateEksTemplateData(&d)
	data.AddValueWithNestedKey("key1", "value1")
	assert.Equal(t, "value1", data.GetStringValueOrDefault("key1", ""))
	data.AddValueWithNestedKey("d1.key2", "a.value2")
	assert.Equal(t, "a.value2", data.Values()["d1"].(map[string]interface{})["key2"].(string))
	data.AddValueWithNestedKey("d1.d2.key3", "b.value3")
	assert.Equal(t, "b.value3", data.Values()["d1"].(map[string]interface{})["d2"].(map[string]interface{})["key3"].(string))
}
