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
	"github.com/aws/aws-sdk-go/service/iam"
	"log"
	"strings"
)

func FindAttachedPolicy(iamClient *iam.IAM, roleName string, policyArn string) (bool, error) {
	var listAttachedRolePoliciesInputMarker *string
	var listAttachedRolePoliciesHasMoreResult bool = true
	for listAttachedRolePoliciesHasMoreResult {
		listAttachedRolePoliciesOutput, listAttachedRolePoliciesErr :=
			iamClient.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
				RoleName: aws.String(roleName),
				Marker:   listAttachedRolePoliciesInputMarker,
			})
		if listAttachedRolePoliciesErr != nil {
			return false, fmt.Errorf("failed to list attached policies for role %s: %v", roleName, listAttachedRolePoliciesErr.Error())
		}
		for _, entry := range listAttachedRolePoliciesOutput.AttachedPolicies {
			if *entry.PolicyArn == policyArn {
				return true, nil
			}
		}
		listAttachedRolePoliciesHasMoreResult = *listAttachedRolePoliciesOutput.IsTruncated
		listAttachedRolePoliciesInputMarker = listAttachedRolePoliciesOutput.Marker
	}
	return false, nil
}

func FindIamPolicy(iamClient *iam.IAM, policyName string) (*iam.Policy, error) {
	var amazonEKSClusterPolicy *iam.Policy

	var listPoliciesInputMarker *string
	var listPoliciesHasMoreResult bool = true
	for listPoliciesHasMoreResult {
		listPoliciesOutput, err := iamClient.ListPolicies(&iam.ListPoliciesInput{
			Marker: listPoliciesInputMarker,
		})
		if err != nil {
			return nil, err
		}
		listPoliciesHasMoreResult = *listPoliciesOutput.IsTruncated
		listPoliciesInputMarker = listPoliciesOutput.Marker
		for _, entry := range listPoliciesOutput.Policies {
			if *entry.PolicyName == policyName {
				amazonEKSClusterPolicy = entry
				break
			}
		}
		if amazonEKSClusterPolicy != nil {
			break
		}
	}

	return amazonEKSClusterPolicy, nil
}

func AttachToRoleByPolicyName(iamClient *iam.IAM, roleName string, policyName string) error {
	policy, err := FindIamPolicy(iamClient, policyName)
	if err != nil {
		return fmt.Errorf("failed to find policy %s: %v", policyName, err.Error())
	}
	if policy == nil {
		return fmt.Errorf("did not find policy %s", policyName)
	}

	return AttachToRoleByPolicyArn(iamClient, roleName, *policy.Arn)
}

func AttachToRoleByPolicyArn(iamClient *iam.IAM, roleName string, policyArn string) error {
	foundAttachedPolicy, err := FindAttachedPolicy(iamClient, roleName, policyArn)
	if err != nil {
		return fmt.Errorf("failed to find policy %s on role %s: %v", policyArn, roleName, err.Error())
	}

	if !foundAttachedPolicy {
		_, err := iamClient.AttachRolePolicy(&iam.AttachRolePolicyInput{
			PolicyArn: aws.String(policyArn),
			RoleName:  aws.String(roleName),
		})
		if err != nil {
			return fmt.Errorf("failed to attach policy %s to role %s: %v", policyArn, roleName, err.Error())
		}
		log.Printf("Attached policy %s to role %s", policyArn, roleName)
	} else {
		log.Printf("Policy %s is already attached role %s, do not attach it again", policyArn, roleName)
	}

	return nil
}

func AttachIamPolicy(iamClient *iam.IAM, roleName string, policyName string, policyDocument string) error {
	createPolicyOutput, err := iamClient.CreatePolicy(&iam.CreatePolicyInput{
		PolicyName:     aws.String(policyName),
		PolicyDocument: aws.String(policyDocument),
	})

	if err != nil {
		if !AlreadyExistsMessage(err.Error()) {
			return fmt.Errorf("failed to create IAM policy %s: %s", policyName, err.Error())
		} else {
			// TODO check policy document and update document if needed
			log.Printf("IAM policy %s already exists, do not create it again", policyName)
		}
	} else {
		log.Printf("Created IAM policy %s", *createPolicyOutput.Policy.PolicyName)
	}

	err = AttachToRoleByPolicyName(iamClient, roleName, policyName)
	if err != nil {
		return fmt.Errorf("failed to attach IAM policy %s to role %s: %s", policyName, roleName, err.Error())
	}

	return nil
}

func GetIamRoleArnByName(region string, roleName string) (string, error) {
	session := CreateSession(region)
	iamClient := iam.New(session)
	getRoleOutput, err := iamClient.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get role %s: %s", roleName, err.Error())
	}
	return *getRoleOutput.Role.Arn, nil
}

func DeleteOidcProvider(region string, oidcProvider string) error {
	httpsPrefix := "https://"
	httpPrefix := "https://"
	oidcProviderLower := strings.ToLower(oidcProvider)
	if strings.HasPrefix(oidcProviderLower, httpsPrefix) {
		oidcProvider = oidcProvider[len(httpsPrefix):]
	} else if strings.HasPrefix(oidcProviderLower, httpPrefix) {
		oidcProvider = oidcProvider[len(httpPrefix):]
	}

	session := CreateSession(region)
	account, err := GetCurrentAccount(session)
	if err != nil {
		return err
	}
	iamClient := iam.New(session)
	arn := fmt.Sprintf("arn:aws:iam::%s:oidc-provider/%s", account, oidcProvider)
	_, err = iamClient.DeleteOpenIDConnectProvider(&iam.DeleteOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: &arn,
	})
	if err != nil {
		return fmt.Errorf("failed to get delete OIDC provider %s: %s", arn, err.Error())
	}
	return nil
}
