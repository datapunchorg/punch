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

package ec2

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/resource"
	"log"
	"strings"
)

func CreateInstances(topologyName string, spec Ec2TopologySpec) ([]*ec2.Instance, error) {
	// TODO, make creating profile as a library, also see https://docs.aws.amazon.com/systems-manager/latest/userguide/setup-instance-profile.html
	roleName := spec.InstanceRole.Name
	profileName := roleName
	session := awslib.CreateSession(spec.Region)
	iamClient := iam.New(session)
	_, err := iamClient.CreateInstanceProfile(&iam.CreateInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == iam.ErrCodeEntityAlreadyExistsException {
				log.Printf("Instance profile %s already exists, do not create it again", profileName)
			} else {
				return nil, fmt.Errorf("failed to create instance profile %s: %s", profileName, err.Error())
			}
		} else {
			return nil, fmt.Errorf("failed to create instance profile %s: %s", profileName, err.Error())
		}
	}
	getInstanceProfileOutput, err := iamClient.GetInstanceProfile(&iam.GetInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get instance profile %s: %s", profileName, err.Error())
	}
	instanceProfile := getInstanceProfileOutput.InstanceProfile
	alreadyContainsRole := false
	for _, role := range instanceProfile.Roles {
		if *role.RoleName == roleName {
			alreadyContainsRole = true
			break
		}
	}
	if !alreadyContainsRole {
		err = resource.CreateIAMRole(spec.Region, spec.InstanceRole)
		if err != nil {
			return nil, fmt.Errorf("failed to create instance IAM role %s: %s", roleName, err.Error())
		}
		_, err = iamClient.AddRoleToInstanceProfile(&iam.AddRoleToInstanceProfileInput{
			InstanceProfileName: aws.String(roleName),
			RoleName:            aws.String(roleName),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to attach IAM role %s to profile %s: %s", roleName, *instanceProfile.Arn, err.Error())
		}
	}
	instanceProfileSpec := ec2.IamInstanceProfileSpecification{
		Arn: instanceProfile.Arn,
	}
	svc := ec2.New(session)
	runResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:            aws.String(spec.ImageId),
		InstanceType:       aws.String(spec.InstanceType),
		MinCount:           aws.Int64(spec.MinCount),
		MaxCount:           aws.Int64(spec.MaxCount),
		IamInstanceProfile: &instanceProfileSpec,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create instace: %s", err)
	}
	var instanceIds []string
	for _, entry := range runResult.Instances {
		instanceIds = append(instanceIds, *entry.InstanceId)
	}
	_, errCreateTags := svc.CreateTags(&ec2.CreateTagsInput{
		Resources: aws.StringSlice(instanceIds),
		Tags: []*ec2.Tag{
			{
				Key:   aws.String(framework.PunchTopologyTagName),
				Value: aws.String(topologyName),
			},
		},
	})
	if errCreateTags != nil {
		return nil, fmt.Errorf("failed to create tags for instances: %s", errCreateTags.Error())
	}
	log.Printf("Created instances in topology: %s", topologyName)
	return runResult.Instances, nil
}

func GetInstanceIdsByTopology(region string, topologyName string) ([]string, error) {
	tagKey := fmt.Sprintf("tag:%s", framework.PunchTopologyTagName)
	tagValue := topologyName
	return GetInstanceIdsByTagKeyValue(region, tagKey, tagValue)
}

func GetInstanceIdsByTagKeyValue(region string, tagKey string, tagValue string) ([]string, error) {
	session := awslib.CreateSession(region)
	svc := ec2.New(session)
	var nextToken *string = nil
	var hasMoreResult bool = true
	var instanceIds []string
	for hasMoreResult {
		describeInstancesOutput, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String(tagKey),
					Values: []*string{aws.String(tagValue)},
				},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get intances for by tag %s=%s: %s", tagKey, tagValue, err.Error())
		}
		for _, reservation := range describeInstancesOutput.Reservations {
			for _, instance := range reservation.Instances {
				instanceIds = append(instanceIds, *instance.InstanceId)
			}
		}
		nextToken = describeInstancesOutput.NextToken
		hasMoreResult = nextToken != nil
	}
	return instanceIds, nil
}

func DeleteInstances(region string, instanceIds []string) error {
	session := awslib.CreateSession(region)
	svc := ec2.New(session)
	_, err := svc.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: aws.StringSlice(instanceIds),
	})
	if err != nil {
		return fmt.Errorf("failed to delete instances: %s", err)
	}
	log.Printf("Deleted insances: %s", strings.Join(instanceIds, ", "))
	return nil
}
