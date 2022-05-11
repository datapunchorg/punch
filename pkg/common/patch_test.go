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
	AnyMapField1 map[string]interface{}
}

type struct4 struct {
	ArrayField []struct3
}

type struct5 struct {
	Field1 struct4
	StructWithStringArrayField structWithStringArrayField
}

type structWithStringArrayField struct {
	StrValues []string
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

	err = PatchValuePathByString(pointer, "'S1'.'Count'", "200")
	assert.Nil(t, err)
	assert.Equal(t, 200, s2.S1.Count)

	err = PatchValuePathByString(pointer, "\"S1\".\"Count\"", "300")
	assert.Nil(t, err)
	assert.Equal(t, 300, s2.S1.Count)
}

func TestPatchStructMapFieldByString(t *testing.T) {
	s3 := struct3{
		MapField1: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key 3": "value3",
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

	err = PatchValuePathByString(pointer, "MapField1.'key 3'", "new_value_3'")
	assert.Nil(t, err)
	assert.Equal(t, "new_value", s3.MapField1["key1"])
	assert.Equal(t, "value2", s3.MapField1["key2"])
	assert.Equal(t, "new_value_3'", s3.MapField1["key 3"])

	err = PatchValuePathByString(pointer, "MapField1.'key 3'", "new_value_3\"")
	assert.Nil(t, err)
	assert.Equal(t, "new_value", s3.MapField1["key1"])
	assert.Equal(t, "value2", s3.MapField1["key2"])
	assert.Equal(t, "new_value_3\"", s3.MapField1["key 3"])
}

func TestPatchStringArray(t *testing.T) {
	v := struct5{
		StructWithStringArrayField: structWithStringArrayField{
			StrValues: []string{"str1", "str2"},
		},
	}

	err := PatchValuePathByString(&v, "StructWithStringArrayField.StrValues[0]", "new_value_111")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(v.StructWithStringArrayField.StrValues))
	assert.Equal(t, "new_value_111", v.StructWithStringArrayField.StrValues[0])
	assert.Equal(t, "str2", v.StructWithStringArrayField.StrValues[1])

	err = PatchValuePathByString(&v, "StructWithStringArrayField.StrValues[1]", "new_value_222")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(v.StructWithStringArrayField.StrValues))
	assert.Equal(t, "new_value_111", v.StructWithStringArrayField.StrValues[0])
	assert.Equal(t, "new_value_222", v.StructWithStringArrayField.StrValues[1])
}

func TestPatchArrayFieldByString(t *testing.T) {
	s5 := struct5{
		Field1: struct4{
			ArrayField: []struct3 {
				{
					MapField1: map[string]string{
						"key.1": "value1",
						"key-2": "value2",
					},
				},
				{
					MapField1: map[string]string{
						"key.11": "value11",
						"key-22": "value22",
					},
					AnyMapField1: map[string]interface{}{
						"keyA": 9,
					},
				},
			},
		},
	}

	err := PatchValuePathByString(&s5, "field1.arrayField[1].MapField1.'key.11'", "new_value_111")
	assert.Nil(t, err)
	assert.Equal(t, "value1", s5.Field1.ArrayField[0].MapField1["key.1"])
	assert.Equal(t, "new_value_111", s5.Field1.ArrayField[1].MapField1["key.11"])

	err = PatchValuePathByString(&s5, "field1.arrayField[1].AnyMapField1.keyA", "99999")
	assert.Nil(t, err)
	assert.Equal(t, 99999, s5.Field1.ArrayField[1].AnyMapField1["keyA"])
}