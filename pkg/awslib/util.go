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

package awslib

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"strings"
)

func CreateSession(region string) *session.Session {
	//creds := credentials.NewEnvCredentials()
	session := session.Must(
		session.NewSession(
			&aws.Config{
				//Credentials: creds,
				Region: aws.String(region),
			}))
	return session
}

func CreateDefaultSession() *session.Session {
	session := session.Must(
		session.NewSession(
			&aws.Config{}))
	return session
}

func GetCurrentAccount(session client.ConfigProvider) (string, error) {
	stsClient := sts.New(session)

	getCallerIdentityOutput, err := stsClient.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}

	return *getCallerIdentityOutput.Account, nil
}

func AlreadyExistsMessage(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "already exist")
}

func SecurityGroupNotFoundMessage(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "invalidgroup.notfound")
}
