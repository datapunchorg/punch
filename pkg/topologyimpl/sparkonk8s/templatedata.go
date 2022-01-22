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

package sparkonk8s

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"log"
)

type SparkTemplateData struct {
	framework.TemplateDataWithVpc
}

func CreateSparkTemplateData(data framework.TemplateData) SparkTemplateData {
	copy := framework.CopyTemplateData(data)
	result := SparkTemplateData{
		TemplateDataWithVpc: framework.TemplateDataWithVpc{
			TemplateDataImpl: copy,
		},
	}
	return result
}

func (t *SparkTemplateData) DefaultS3BucketName() string {
	namePrefix := t.NamePrefix()
	region := t.TemplateDataWithVpc.Region()

	session := awslib.CreateDefaultSession()
	account, err := awslib.GetCurrentAccount(session)
	if err != nil {
		log.Printf("[WARN] Failed to get current AWS account: %s", err.Error())
		return "error-no-value"
	}

	return fmt.Sprintf("%s-%s-%s", namePrefix, account, region)
}


