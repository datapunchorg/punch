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
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/datapunchorg/punch/pkg/awslib"
	"log"
)

type SecurityGroup struct {
	Name string `json:"name" yaml:"name"`
	InboundRules []SecurityGroupInboundRule `json:"inboundRules" yaml:"inboundRules"`
}

type SecurityGroupInboundRule struct {
	IPProtocol string `json:"ipProtocol" yaml:"ipProtocol"`
	FromPort int64 `json:"fromPort" yaml:"fromPort"`
	ToPort int64 `json:"toPort" yaml:"toPort"`
	IPRanges []string `json:"ipRanges" yaml:"ipRanges"`
}

func CreateSecurityGroup(ec2Client *ec2.EC2, securityGroup SecurityGroup, vpcId string) (string, error) {
	securityGroupName := securityGroup.Name
	createSecurityGroupResult, err := ec2Client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(securityGroupName),
		Description: aws.String(securityGroupName),
		VpcId:       aws.String(vpcId),
	})
	if err != nil {
		if !awslib.AlreadyExistsMessage(err.Error()) {
			return "", fmt.Errorf("failed to create security group %s: %s", securityGroupName, err.Error())
		} else {
			log.Printf("Find and use existing security group %s", securityGroupName)
		}
	} else {
		log.Printf("Created security group %s, id: %v", securityGroupName, createSecurityGroupResult.GroupId)
	}

	describeSecurityGroupsOutput, err := ec2Client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("group-name"),
				Values: []*string{
					aws.String(securityGroupName),
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get security group %s: %s", securityGroupName, err.Error())
	}
	if len(describeSecurityGroupsOutput.SecurityGroups) == 0 {
		return "", fmt.Errorf("got empty result for security group %s: %s", securityGroupName, err.Error())
	}
	if *(describeSecurityGroupsOutput.SecurityGroups[0].GroupName) != securityGroupName {
		return "", fmt.Errorf("got unexpected security group %s when trying to get %s", *(describeSecurityGroupsOutput.SecurityGroups[0].GroupName), securityGroupName)
	}

	securityGroupId := describeSecurityGroupsOutput.SecurityGroups[0].GroupId

	if securityGroupId == nil {
		return "", fmt.Errorf("id is empty for security group %s", *(describeSecurityGroupsOutput.SecurityGroups[0].GroupName))
	}

	for _, rule := range securityGroup.InboundRules {
		ipRanges := make([]*ec2.IpRange, 0, 10)
		for _, ip := range rule.IPRanges {
			ipRange := ec2.IpRange{CidrIp: aws.String(ip)}
			ipRanges = append(ipRanges, &ipRange)
		}
		_, err = ec2Client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
			GroupId: securityGroupId,
			IpPermissions: []*ec2.IpPermission{
				(&ec2.IpPermission{}).
					SetIpProtocol(rule.IPProtocol).
					SetFromPort(rule.FromPort).
					SetToPort(rule.ToPort).
					SetIpRanges(ipRanges),
			},
		})
		if err != nil {
			if !awslib.AlreadyExistsMessage(err.Error()) {
				return "", fmt.Errorf("unable to set ingress permission for security group %s: %s", securityGroupName, err.Error())
			} else {
				log.Printf("Ingress permission already set for security group %s, do not set it again", securityGroupName)
			}
		} else {
			log.Printf("Set ingress permission for security group %s", securityGroupName)
		}
	}
	return *securityGroupId, nil
}

