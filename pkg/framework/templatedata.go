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
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/datapunchorg/punch/pkg/awslib"
	"log"
	"strings"
)

const (
	TemplateNoValue   = "<no value>"
)

type TemplateData interface {
	Values() map[string]interface{}
}

func CreateTemplateData(values map[string]string) TemplateDataImpl {
	result := TemplateDataImpl{
		values: map[string]interface{}{},
	}
	for key, value := range values {
		result.AddValueWithNestedKey(key, value)
	}
	return result
}

func CopyTemplateData(from TemplateData) TemplateDataImpl {
	result := TemplateDataImpl{
		values: map[string]interface{}{},
	}
	for key, value := range from.Values() {
		result.values[key] = value
	}
	return result
}

type TemplateDataImpl struct {
	values map[string]interface{}
}

func (t *TemplateDataImpl) Values() map[string]interface{} {
	return t.values
}

func (t *TemplateDataImpl) AddValue(key string, value interface{}) {
	t.values[key] = value
}

func (t *TemplateDataImpl) AddValues(values map[string]interface{}) {
	for key, value := range values {
		t.values[key] = value
	}
}

func (t *TemplateDataImpl) AddValueWithNestedKey(key string, value interface{}) {
	parts := strings.Split(key, ".")
	addDataValueWithNestedKey(t.values, parts, value)
}

func (t *TemplateDataImpl) NamePrefix() string {
	return t.GetStringValueOrDefault("namePrefix", DefaultNamePrefix)
}

func (t *TemplateDataImpl) GetStringValueOrDefault(key string, defaultValue string) string {
	v := t.Values()[key]
	if v == nil {
		v = defaultValue
	}
	return v.(string)
}

func addDataValueWithNestedKey(data map[string]interface{}, key []string, value interface{}) {
	if len(key) == 0 {
		panic(fmt.Errorf("cannot add empty key into data"))
	}
	if len(key) == 1 {
		data[key[0]] = value
	} else {
		entry, ok := data[key[0]]
		if !ok {
			entry = map[string]interface{}{}
			data[key[0]] = entry
		}
		newData := entry.(map[string]interface{})
		newKey := key[1:]
		addDataValueWithNestedKey(newData, newKey, value)
	}
}

type TemplateDataWithRegion struct {
	TemplateDataImpl
}

func (t *TemplateDataWithRegion) Region() string {
	return t.GetStringValueOrDefault("region", DefaultRegion)
}

func (t *TemplateDataWithRegion) DefaultVpcId() string {
	region := t.Region()
	session := awslib.CreateSession(region)
	ec2Client := ec2.New(session)

	vpcId, err := awslib.GetFirstVpcId(ec2Client)
	if err != nil {
		log.Printf("[WARN] Failed to get first VPC: %s", err.Error())
		return "error-no-value"
	}

	return vpcId
}

func (t *TemplateDataWithRegion) DefaultS3BucketName() string {
	namePrefix := t.NamePrefix()
	region := t.Region()

	session := awslib.CreateDefaultSession()
	account, err := awslib.GetCurrentAccount(session)
	if err != nil {
		log.Printf("[WARN] Failed to get current AWS account: %s", err.Error())
		return "error-no-value"
	}

	return fmt.Sprintf("%s-%s-%s", namePrefix, account, region)
}

func CreateTemplateDataWithRegion(data TemplateData) TemplateDataWithRegion {
	copy := CopyTemplateData(data)
	return TemplateDataWithRegion{
		TemplateDataImpl: copy,
	}
}
