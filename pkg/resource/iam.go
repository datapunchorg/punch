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

package resource

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/datapunchorg/punch/pkg/awslib"
	"log"
)

type IamRole struct {
	Name string `json:"name" yaml:"name"`
	// The trust relationship policy document that grants an entity permission to assume the role.
	AssumeRolePolicyDocument string      `json:"assumeRolePolicyDocument" yaml:"assumeRolePolicyDocument"`
	Policies                 []IamPolicy `json:"policies" yaml:"policies"`
	ExtraPolicyArns          []string    `json:"extraPolicyArns" yaml:"extraPolicyArns"`
}

type IamPolicy struct {
	Name           string `json:"name" yaml:"name"`
	PolicyDocument string `json:"policyDocument" yaml:"policyDocument"`
}

func CreateIamRole(region string, iamRole IamRole) error {
	session := awslib.CreateSession(region)
	iamClient := iam.New(session)
	assumeRolePolicyDocument := iamRole.AssumeRolePolicyDocument
	roleName := iamRole.Name
	createRoleResult, err := iamClient.CreateRole(&iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(assumeRolePolicyDocument),
	})
	if err != nil {
		if !awslib.AlreadyExistsMessage(err.Error()) {
			return fmt.Errorf("failed to create IAM role %s: %s", roleName, err.Error())
		} else {
			log.Printf("IAM role %s already exists, do not create it again", roleName)
		}
	} else {
		log.Printf("Created IAM role %s", *createRoleResult.Role.RoleName)
	}

	for _, entry := range iamRole.Policies {
		err = awslib.AttachIamPolicy(iamClient, roleName, entry.Name, entry.PolicyDocument)
		if err != nil {
			return fmt.Errorf("failed to to attach IAM policy %s to role %s: %s", entry.Name, roleName, err.Error())
		}
	}

	for _, entry := range iamRole.ExtraPolicyArns {
		err = awslib.AttachToRoleByPolicyArn(iamClient, roleName, entry)
		if err != nil {
			return fmt.Errorf("failed to attach IAM policy %s to role %s: %s", entry, roleName, err.Error())
		}
	}

	return nil
}

func CreateIamRoleWithMorePolicies(region string, iamRole IamRole, policies []IamPolicy) (string, error) {
	session := awslib.CreateSession(region)
	iamClient := iam.New(session)

	err := CreateIamRole(region, iamRole)
	if err != nil {
		return "", fmt.Errorf("failed to create IAM role: %s", err.Error())
	}

	roleName := iamRole.Name
	for _, policy := range policies {
		policyName := policy.Name
		policyDocument := policy.PolicyDocument
		err = awslib.AttachIamPolicy(iamClient, roleName, policyName, policyDocument)
		if err != nil {
			return "", fmt.Errorf("failed to to attach IAM policy %s to role %s: %v", policyName, roleName, err)
		}
	}

	return roleName, nil
}
