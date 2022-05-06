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
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type pathSegmentType byte

var pathSegmentTypeField = pathSegmentType(0)
var pathSegmentTypeArray = pathSegmentType(1)

type pathSegment struct {
	segmentType pathSegmentType
	name string
	arrayIndex int
}

func PatchValuePathByString(target interface{}, path string, value string) error {
	parts, err := parsePath(path)
	if err != nil {
		return err
	}
	return patchStructPathArrayByStringValue(target, parts, value)
}

func PatchValueFieldByString(target interface{}, field string, value string) error {
	valueOfTarget := reflect.ValueOf(target).Elem()
	if valueOfTarget.Kind() == reflect.Struct {
		return patchStructFieldByStringValue(target, field, value)
	} else if valueOfTarget.Kind() == reflect.Map {
		return patchMapKeyByStringValue(target, field, value)
	} else {
		return fmt.Errorf("cannot patch due to unsupported target value type: %s", valueOfTarget.Kind())
	}
}

func patchStructFieldByStringValue(target interface{}, field string, value string) error {
	fieldV, err := getFieldByName(target, field)
	if err != nil {
		return err
	}
	switch fieldV.Kind() {
	case reflect.String:
		fieldV.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intV, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("field %s is integer, but value %s cannot be parsed to integer", field, value)
		}
		fieldV.SetInt(intV)
	case reflect.Float32, reflect.Float64:
		floatV, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("field %s is float, but value %s cannot be parsed to float", field, value)
		}
		fieldV.SetFloat(floatV)
	case reflect.Bool:
		boolV, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("field %s is bool, but value %s cannot be parsed to bool", field, value)
		}
		fieldV.SetBool(boolV)
	default:
		return fmt.Errorf("cannot set value for field %s due to unsupported type: %s", field, fieldV.Kind())
	}
	return nil
}

func convertStringValue(value string, kind reflect.Kind) (reflect.Value, error) {
	switch kind {
	case reflect.String:
		return reflect.ValueOf(value), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intV, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("kind %s is integer, but value %s cannot be parsed to integer", kind, value)
		}
		switch kind {
		case reflect.Int:
			return reflect.ValueOf(int(intV)), nil
		case reflect.Int8:
			return reflect.ValueOf(int8(intV)), nil
		case reflect.Int16:
			return reflect.ValueOf(int16(intV)), nil
		case reflect.Int32:
			return reflect.ValueOf(int32(intV)), nil
		case reflect.Int64:
			return reflect.ValueOf(int64(intV)), nil
		default:
			return reflect.Value{}, fmt.Errorf("kind %s is integer, but not handled", kind)
		}
	case reflect.Float32, reflect.Float64:
		floatV, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("kind %s is float, but value %s cannot be parsed to float", kind, value)
		}

		switch kind {
		case reflect.Float32:
			return reflect.ValueOf(float32(floatV)), nil
		case reflect.Float64:
			return reflect.ValueOf(float64(floatV)), nil
		default:
			return reflect.Value{}, fmt.Errorf("kind %s is float, but not handled", kind)
		}
	case reflect.Bool:
		boolV, err := strconv.ParseBool(value)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("kind %s is bool, but value %s cannot be parsed to bool", kind, value)
		}
		return reflect.ValueOf(boolV), nil
	default:
		return reflect.Value{}, fmt.Errorf("cannot convert value for field %s due to unsupported type: %s", value, kind)
	}
}

func patchMapKeyByStringValue(target interface{}, field string, value string) error {
	valueOfTarget := reflect.ValueOf(target).Elem()
	if valueOfTarget.Kind() == reflect.Map {
		//if mapElementType.Kind() != reflect.String {
		//	return fmt.Errorf("target is map, but type for the map value is not string: %s", mapElementType.Kind())
		//}
		for _, key := range valueOfTarget.MapKeys() {
			keyStr := key.String()
			if strings.EqualFold(keyStr, field) {
				elementValue := valueOfTarget.MapIndex(key)
				elementValueKind := elementValue.Type().Kind()
				if elementValueKind == reflect.Interface {
					elementValueKind = elementValue.Elem().Type().Kind()
				}
				newValue, err := convertStringValue(value, elementValueKind)
				if err != nil {
					return fmt.Errorf("cannot convert string %s to map element kind %s", value, elementValueKind.String())
				}
				valueOfTarget.SetMapIndex(key, newValue)
				return nil
			}
		}
		return fmt.Errorf("key %s does not exist on target map", field)
	}
	return fmt.Errorf("target value is not a pointer to map")
}

func getFieldByName(target interface{}, field string) (reflect.Value, error) {
	valueOfTarget := reflect.ValueOf(target).Elem()
	if valueOfTarget.Kind() == reflect.Struct {
		typeOfTarget := valueOfTarget.Type()
		var fieldV reflect.Value
		var found = false
		for i := 0; i < typeOfTarget.NumField(); i++ {
			fieldT := typeOfTarget.Field(i)
			fieldName := fieldT.Name
			if strings.EqualFold(fieldName, field) {
				fieldV = valueOfTarget.Field(i)
				found = true
				break
			}
		}
		if !found {
			return reflect.Value{}, fmt.Errorf("field %s does not exist on target struct", field)
		}
		return fieldV, nil
	} else if valueOfTarget.Kind() == reflect.Map {
		for _, key := range valueOfTarget.MapKeys() {
			keyStr := key.String()
			if strings.EqualFold(keyStr, field) {
				return valueOfTarget.MapIndex(key), nil
			}
		}
		return reflect.Value{}, fmt.Errorf("key %s does not exist on target map", field)
	} else {
		return reflect.Value{}, fmt.Errorf("target value is not a pointer to struct or map")
	}
}

func patchStructPathArrayByStringValue(target interface{}, path []pathSegment, value string) error {
	if len(path) == 0 {
		return fmt.Errorf("invalid path (zero length)")
	}
	segment := path[0]
	if len(path) == 1 {
		if segment.segmentType == pathSegmentTypeField {
			return PatchValueFieldByString(target, segment.name, value)
		} else {
			return fmt.Errorf("TODO unsupported path segment type %d", segment.segmentType)
		}
	}
	if segment.segmentType == pathSegmentTypeField {
		fieldV, err := getFieldByName(target, segment.name)
		if err != nil {
			return err
		}
		p := fieldV.Addr().Interface()
		return patchStructPathArrayByStringValue(p, path[1:], value)
	} else if segment.segmentType == pathSegmentTypeArray {
		fieldV, err := getFieldByName(target, segment.name)
		if err != nil {
			return err
		}
		if fieldV.Kind() == reflect.Slice || fieldV.Kind() == reflect.Array {
			p := fieldV.Index(segment.arrayIndex).Addr().Interface()
			return patchStructPathArrayByStringValue(p, path[1:], value)
		} else {
			return fmt.Errorf("value is not slice/array, cannot patch by %s[%d]", segment.name, segment.arrayIndex)
		}
	} else {
		return fmt.Errorf("unsupported path segment type %d", segment.segmentType)
	}
}


func parsePath(path string) ([]pathSegment, error) {
	segments := make([]pathSegment, 0, 100)
	stateSegmentStart := 0
	statePlainField := 1
	stateQuotedField := 2
	stateSliceBracketStart := 3
	stateSliceBracketEnd := 4
	stateQuotedFieldEnd := 5
	state := stateSegmentStart
	nameCollector := make([]rune, 0, 100)
	arrayIndexCollector := make([]rune, 0, 100)
	quote := '\''
	characters := make([]rune, 0, len(path))
	for _, ch := range path {
		characters = append(characters, ch)
	}
	characters = append(characters, -1)
	for position, ch := range characters {
		switch state {
		case stateSegmentStart:
			if ch == '\'' || ch == '"' {
				state = stateQuotedField
				quote = ch
			} else if ch == '.' {
				return nil, fmt.Errorf("invalid path %s (unexpected .) at position (starting from 0) %d", path, position)
			} else if ch == -1 {
				return nil, fmt.Errorf("invalid path %s (unexpected string end) at position (starting from 0) %d", path, position)
			} else {
				state = statePlainField
				nameCollector = append(nameCollector, ch)
			}
		case statePlainField:
			if ch == '.' || ch == -1 {
				if len(nameCollector) == 0 {
					return nil, fmt.Errorf("invalid path %s (empty segment name) at position (starting from 0) %d", path, position)
				}
				segments = append(segments, pathSegment{
					segmentType: pathSegmentTypeField,
					name: string(nameCollector),
				})
				state = stateSegmentStart
				nameCollector = make([]rune, 0, 100)
			} else if ch == '[' {
				state = stateSliceBracketStart
				arrayIndexCollector = make([]rune, 0, 100)
			} else {
				nameCollector = append(nameCollector, ch)
			}
		case stateSliceBracketStart:
			if ch == ']' {
				state = stateSliceBracketEnd
			} else if ch == -1 {
				return nil, fmt.Errorf("invalid path %s (unexpected string end) at position (starting from 0) %d", path, position)
			} else {
				arrayIndexCollector = append(arrayIndexCollector, ch)
			}
		case stateSliceBracketEnd:
			if ch == '.' || ch == -1 {
				if len(nameCollector) == 0 {
					return nil, fmt.Errorf("invalid path %s (empty segment name) at position (starting from 0) %d", path, position)
				}
				indexStr := string(arrayIndexCollector)
				index, err := strconv.Atoi(indexStr)
				if err != nil {
					return nil, fmt.Errorf("invalid path %s (invalid array index %s) at position (starting from 0) %d", path, indexStr, position)
				}
				segments = append(segments, pathSegment{
					segmentType: pathSegmentTypeArray,
					name: string(nameCollector),
					arrayIndex: index,
				})
				state = stateSegmentStart
				nameCollector = make([]rune, 0, 100)
			} else {
				return nil, fmt.Errorf("invalid path %s (unexpected character after array bracket end) at position (starting from 0) %d", path, position)
			}
		case stateQuotedField:
			if ch == quote {
				state = stateQuotedFieldEnd
			} else if ch == -1 {
				return nil, fmt.Errorf("invalid path %s (unexpected string end) at position (starting from 0) %d", path, position)
			} else {
				nameCollector = append(nameCollector, ch)
			}
		case stateQuotedFieldEnd:
			if ch == '.' || ch == -1 {
				if len(nameCollector) == 0 {
					return nil, fmt.Errorf("invalid path %s (empty segment name) at position (starting from 0) %d", path, position)
				}
				segments = append(segments, pathSegment{
					segmentType: pathSegmentTypeField,
					name: string(nameCollector),
				})
				state = stateSegmentStart
				nameCollector = make([]rune, 0, 100)
			} else {
				return nil, fmt.Errorf("invalid path %s (unexpected character after quoted field end) at position (starting from 0) %d", path, position)
			}
		default:
			return nil, fmt.Errorf("invalid state %d when parsing path %s", state, path)
		}
	}
	return segments, nil
}