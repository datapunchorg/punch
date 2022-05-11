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

package awslib

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"net/url"
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

func CreateSessionOrError(region string) (*session.Session, error) {
	session, err := session.NewSession(
		&aws.Config{
			Region: aws.String(region),
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session for region %s: %s", region, err.Error())
	}
	return session, nil
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

func GetS3BucketAndKeyFromUrl(str string) (string, string, error) {
	u, err := url.Parse(str)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse S3 url %s: %s", str, err.Error())
	}
	bucket := u.Host
	if bucket == "" {
		return "", "", fmt.Errorf("invalid S3 url %s", str)
	}
	key := u.Path
	if key == "" || key == "/" {
		return "", "", fmt.Errorf("invalid S3 url %s", str)
	}
	if strings.HasPrefix(key, "/") {
		key = key[1:]
	}
	return bucket, key, nil
}