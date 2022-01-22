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
	"github.com/aws/aws-sdk-go/service/ec2"
	"log"
)

func GetFirstVpcId(ec2Client *ec2.EC2) (string, error) {
	describeVpcsResult, err := ec2Client.DescribeVpcs(nil)
	if err != nil {
		return "", fmt.Errorf("failed to get VPCs: %s", err.Error())
	}

	if len(describeVpcsResult.Vpcs) == 0 {
		return "", fmt.Errorf("did not find any VPCs. Please add a VPC in your AWS account")
	}

	return *describeVpcsResult.Vpcs[0].VpcId, nil
}

func CheckOrGetFirstVpcId(ec2Client *ec2.EC2, vpcId string) (string, error) {
	var err error
	if vpcId == "" {
		vpcId, err = GetFirstVpcId(ec2Client)
		if err != nil {
			return "", fmt.Errorf("failed to get first VPC: %s", err.Error())
		}
		log.Printf("Find and use first VPC: %s", vpcId)
		return vpcId, nil
	} else {
		log.Printf("Use provided VPC: %s", vpcId)
		return vpcId, nil
	}
}