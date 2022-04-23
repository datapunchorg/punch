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
	"time"
)

func RetryUntilTrue(run func() (bool, error), maxWait time.Duration, sleepTime time.Duration) error {
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
			time.Sleep(sleepTime)
		}
		currentTime = time.Now()
	}
	return fmt.Errorf("timed out after running %d seconds while max wait time is %d seconds", int(currentTime.Sub(startTime).Seconds()), int(maxWait.Seconds()))
}

func PatchStructFieldByStringValue(target interface{}, field string, value string) error {
	valueOfTarget := reflect.ValueOf(target).Elem()
	typeOfTarget := valueOfTarget.Type()
	if valueOfTarget.Kind() != reflect.Struct {
		return fmt.Errorf("target value is not a pointer to struct")
	}
	var fieldV reflect.Value
	var found = false
	for i := 0; i < typeOfTarget.NumField(); i++ {
		fieldT := typeOfTarget.Field(i)
		fieldName := fieldT.Name
		if fieldName == field {
			fieldV = valueOfTarget.Field(i)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("field %s does not exist on target struct", field)
	}
	if !fieldV.CanSet() {
		return fmt.Errorf("field %s cannot set on target struct", field)
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
	}
	return nil
}