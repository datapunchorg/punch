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

package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type struct1 struct {
	Id string
	Enabled bool
	Count int
}

type struct2 struct {
	field1 string
	S1 struct1
}

type struct3 struct {
	MapField1 map[string]string
}

func TestPatchStructFieldByString(t *testing.T) {
	s1 := struct1{}
	var pointer interface{} = &s1
	err := PatchValueFieldByString(pointer, "Id123", "new_value")
	assert.NotNil(t, err)
	err = PatchValueFieldByString(pointer, "Id", "new_value")
	assert.Nil(t, err)
	assert.Equal(t, "new_value", s1.Id)
	err = PatchValueFieldByString(pointer, "Enabled", "true")
	assert.Nil(t, err)
	assert.Equal(t, true, s1.Enabled)
	err = PatchValueFieldByString(pointer, "Enabled", "false")
	assert.Nil(t, err)
	assert.Equal(t, false, s1.Enabled)
	err = PatchValueFieldByString(pointer, "Count", "100")
	assert.Nil(t, err)
	assert.Equal(t, 100, s1.Count)
}

func TestPatchStructPathByString(t *testing.T) {
	s1 := struct1{}
	var pointer interface{} = &s1
	err := PatchValuePathByString(pointer, "Id123", "new_value")
	assert.NotNil(t, err)
	err = PatchValuePathByString(pointer, "Id", "new_value")
	assert.Nil(t, err)
	assert.Equal(t, "new_value", s1.Id)
	err = PatchValuePathByString(pointer, "Enabled", "true")
	assert.Nil(t, err)
	assert.Equal(t, true, s1.Enabled)
	err = PatchValuePathByString(pointer, "Enabled", "false")
	assert.Nil(t, err)
	assert.Equal(t, false, s1.Enabled)
	err = PatchValuePathByString(pointer, "Count", "100")
	assert.Nil(t, err)
	assert.Equal(t, 100, s1.Count)

	s2 := struct2{}
	pointer = &s2
	err = PatchValuePathByString(pointer, "s1.Id", "new_value")
	assert.Nil(t, err)
	assert.Equal(t, "new_value", s2.S1.Id)
	err = PatchValuePathByString(pointer, "S1.enabled", "true")
	assert.Nil(t, err)
	assert.Equal(t, true, s2.S1.Enabled)
	err = PatchValuePathByString(pointer, "S1.Enabled", "false")
	assert.Nil(t, err)
	assert.Equal(t, false, s2.S1.Enabled)
	err = PatchValuePathByString(pointer, "S1.Count", "100")
	assert.Nil(t, err)
	assert.Equal(t, 100, s2.S1.Count)
}

func TestPatchStructMapFieldByString(t *testing.T) {
	s3 := struct3{
		MapField1: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}
	var pointer interface{} = &s3
	err := PatchValuePathByString(pointer, "MapField1", "new_value")
	assert.NotNil(t, err)
	err = PatchValuePathByString(pointer, "MapField1.not_existing_key", "new_value")
	assert.NotNil(t, err)
	err = PatchValuePathByString(pointer, "MapField1.key1", "new_value")
	assert.Nil(t, err)
	assert.Equal(t, "new_value", s3.MapField1["key1"])
	assert.Equal(t, "value2", s3.MapField1["key2"])
}
