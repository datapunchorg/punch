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
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/datapunchorg/punch/pkg/common"
	"log"
	"time"
)

func GetSecurityGroupId(ec2Client *ec2.EC2, vpcId string, securityGroupName string) (string, error) {
	list, err := ListSecurityGroups(ec2Client, vpcId, securityGroupName)
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", fmt.Errorf("did not find security group %s in vpc %s", securityGroupName, vpcId)
	}
	if len(list) > 1 {
		return "", fmt.Errorf("found multiple security groups by name %s in vpc %s", securityGroupName, vpcId)
	}
	first := list[0]
	if first == nil {
		return "", fmt.Errorf("did not find security group %s in vpc %s (nil pointer)", securityGroupName, vpcId)
	}
	return *first.GroupId, nil
}

func ListSecurityGroups(ec2Client *ec2.EC2, vpcId string, securityGroupName string) ([]*ec2.SecurityGroup, error) {
	var nextToken *string = nil
	var hasMoreResult bool = true
	var result []*ec2.SecurityGroup = nil
	for hasMoreResult {
		describeSecurityGroupsOutput, err := ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("vpc-id"),
					Values: []*string{
						aws.String(vpcId),
					},
				},
				{
					Name: aws.String("group-name"),
					Values: []*string{
						aws.String(securityGroupName),
					},
				},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list security groups by vpc %s and name %s: %s", vpcId, securityGroupName, err.Error())
		}
		result = append(result, describeSecurityGroupsOutput.SecurityGroups...)
		nextToken = describeSecurityGroupsOutput.NextToken
		hasMoreResult = nextToken != nil
	}
	return result, nil
}

func DeleteSecurityGroupById(ec2Client *ec2.EC2, securityGroupId string) error {
	_, err := ec2Client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(securityGroupId),
	})
	if err != nil {
		return fmt.Errorf("failed to delete security group %s: %s", securityGroupId, err.Error())
	}
	return nil
}

func ListSecurityGroupRoles(ec2Client *ec2.EC2, vpcId string, securityGroupName string) ([]*ec2.SecurityGroupRule, error) {
	var nextToken *string = nil
	var hasMoreResult bool = true
	var result []*ec2.SecurityGroupRule = nil
	for hasMoreResult {
		describeOutput, err := ec2Client.DescribeSecurityGroupRules(&ec2.DescribeSecurityGroupRulesInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("vpc-id"),
					Values: []*string{
						aws.String(vpcId),
					},
				},
				{
					Name: aws.String("group-name"),
					Values: []*string{
						aws.String(securityGroupName),
					},
				},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list security groups by vpc %s and name %s: %s", vpcId, securityGroupName, err.Error())
		}
		result = append(result, describeOutput.SecurityGroupRules...)
		nextToken = describeOutput.NextToken
		hasMoreResult = nextToken != nil
	}
	return result, nil
}

func ListNetworkInterfaces(ec2Client *ec2.EC2, vpcId string, securityGroupId string) ([]*ec2.NetworkInterface, error) {
	var nextToken *string = nil
	var hasMoreResult bool = true
	var result []*ec2.NetworkInterface = nil
	for hasMoreResult {
		describeOutput, err := ec2Client.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("vpc-id"),
					Values: []*string{
						aws.String(vpcId),
					},
				},
				{
					Name: aws.String("group-id"),
					Values: []*string{
						aws.String(securityGroupId),
					},
				},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list network interfaces by vpc %s and security group id %s: %s", vpcId, securityGroupId, err.Error())
		}
		result = append(result, describeOutput.NetworkInterfaces...)
		nextToken = describeOutput.NextToken
		hasMoreResult = nextToken != nil
	}
	return result, nil
}

func DeleteSecurityGroupByIdIgnoreError(ec2Client *ec2.EC2, securityGroupId string, maxRetryDuration time.Duration) {
	err := common.RetryUntilTrue(func() (bool, error) {
		err := DeleteSecurityGroupById(ec2Client, securityGroupId)
		if err != nil {
			if SecurityGroupNotFoundMessage(err.Error()) {
				log.Printf("Security group %s already deleted, do not delete it again", securityGroupId)
				return true, nil
			} else {
				log.Printf("[WARN] Failed to delete security group %s: %s. Retrying...", securityGroupId, err.Error())
				return false, nil
			}
		} else {
			log.Printf("Deleted security group %s", securityGroupId)
			return true, nil
		}
	},
		maxRetryDuration,
		10*time.Second)

	if err != nil {
		log.Printf("[WARN] Failed to delete security group %s after retrying: %s", securityGroupId, err.Error())
	}
}

func DeleteNodeGroup(region string, clusterName string, nodeGroupName string) error {
	session := CreateSession(region)

	log.Printf("Deleting node group %s from Eks cluster %s in AWS region: %v", nodeGroupName, clusterName, region)

	eksClient := eks.New(session)

	exists, err := NodeGroupExists(eksClient, clusterName, nodeGroupName)
	if err != nil {
		log.Printf("Failed to check whether node gorup %s exists in cluster %s: %s", nodeGroupName, clusterName, err.Error())
	}

	if !exists {
		log.Printf("Node group %s does not exist in cluster %s, no need to delete", nodeGroupName, clusterName)
		return nil
	}

	log.Printf("Deleting node group %s in cluster %s", nodeGroupName, clusterName)

	_, err = eksClient.DeleteNodegroup(&eks.DeleteNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: aws.String(nodeGroupName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete node group %s from Eks cluster %s: %s", nodeGroupName, clusterName, err.Error())
	}

	waitDeletedErr := common.RetryUntilTrue(func() (bool, error) {
		stillExists, err := NodeGroupExists(eksClient, clusterName, nodeGroupName)
		if err != nil {
			return false, err
		}
		if stillExists {
			log.Printf("Node group %s still exists, waiting...", nodeGroupName)
		} else {
			log.Printf("Node group %s does not exist", nodeGroupName)
		}
		return !stillExists, nil
	}, 10*time.Minute, 30*time.Second)

	if waitDeletedErr != nil {
		return fmt.Errorf("failed to wait node group %s to be deleted: %s", nodeGroupName, err.Error())
	}

	return nil
}

func GetDefaultVpcId(region string) (string, error) {
	session := CreateSession(region)
	ec2Client := ec2.New(session)

	vpcId, err := GetFirstVpcId(ec2Client)
	if err != nil {
		return "", err
	}

	return vpcId, nil
}