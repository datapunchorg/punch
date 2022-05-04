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
	"time"
)

type HostPort struct {
	Host string
	Port int32
}

func RetryUntilTrue(run func() (bool, error), maxWait time.Duration, retryInterval time.Duration) error {
	currentTime := time.Now()
	startTime := currentTime
	endTime := currentTime.Add(maxWait)
	for !currentTime.After(endTime) {
		result, err := run()
		if err != nil {
			return err
		}
		if result {
			return nil
		}
		if !currentTime.After(endTime) {
			time.Sleep(retryInterval)
		}
		currentTime = time.Now()
	}
	return fmt.Errorf("timed out after running %d seconds while max wait time is %d seconds", int(currentTime.Sub(startTime).Seconds()), int(maxWait.Seconds()))
}

func PatchValuePathByString(target interface{}, path string, value string) error {
	parts := strings.Split(path, ".")
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
	fieldV, err := getSettableField(target, field)
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

func patchMapKeyByStringValue(target interface{}, field string, value string) error {
	valueOfTarget := reflect.ValueOf(target).Elem()
	newValue := reflect.ValueOf(value)
	if valueOfTarget.Kind() == reflect.Map {
		typeOfTarget := valueOfTarget.Type()
		mapElementType := typeOfTarget.Elem()
		if mapElementType.Kind() != reflect.String {
			return fmt.Errorf("target is map, but type for the map value is not string: %s", mapElementType.Kind())
		}
		for _, key := range valueOfTarget.MapKeys() {
			keyStr := key.String()
			if strings.EqualFold(keyStr, field) {
				valueOfTarget.SetMapIndex(key, newValue)
				return nil
			}
		}
		return fmt.Errorf("key %s does not exist on target map", field)
	}
	return fmt.Errorf("target value is not a pointer to struct")
}

func getSettableField(target interface{}, field string) (reflect.Value, error) {
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
		if !fieldV.CanSet() {
			return reflect.Value{}, fmt.Errorf("field %s cannot set on target struct", field)
		}
		return fieldV, nil
	} else {
		return reflect.Value{}, fmt.Errorf("target value is not a pointer to struct")
	}
}

func patchStructPathArrayByStringValue(target interface{}, path []string, value string) error {
	if len(path) == 0 {
		return fmt.Errorf("invalid path (zero length)")
	} else if len(path) == 1 {
		return PatchValueFieldByString(target, path[0], value)
	}
	fieldV, err := getSettableField(target, path[0])
	if err != nil {
		return err
	}
	p := fieldV.Addr().Interface()
	return patchStructPathArrayByStringValue(p, path[1:], value)
}
