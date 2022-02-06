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
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/common"
	"log"
	"strings"
	"time"
)

type EKSCluster struct {
	ClusterName string `json:"clusterName" yaml:"clusterName"`
	// The Amazon Resource Name (ARN) of the IAM role that provides permissions
	// for the Kubernetes control plane to make calls to Amazon Web Services API
	// operations on your behalf. For more information, see Amazon EKS Service IAM
	// Role (https://docs.aws.amazon.com/eks/latest/userguide/service_IAM_role.html)
	// in the Amazon EKS User Guide.
	ControlPlaneRole IAMRole `json:"controlPlaneRole" yaml:"controlPlaneRole"`
	// See https://docs.aws.amazon.com/eks/latest/userguide/sec-group-reqs.html
	SecurityGroups   []SecurityGroup        `json:"securityGroups" yaml:"securityGroups"`
	InstanceRole     IAMRole                `json:"instanceRole" yaml:"instanceRole"`
}


func CreateEksCluster(region string, vpcId string, eksCluster EKSCluster) error {
	clusterName := eksCluster.ClusterName

	session := awslib.CreateSession(region)

	log.Printf("Creating EKS cluster in AWS region: %v", region)

	ec2Client := ec2.New(session)

	var err error

	if vpcId == "" {
		vpcId, err = awslib.GetFirstVpcId(ec2Client)
		if err != nil {
			return fmt.Errorf("failed to get first VPC: %s", err.Error())
		}
		log.Printf("Find and use first VPC: %s", vpcId)
	} else {
		log.Printf("Use provided VPC: %s", vpcId)
	}

	securityGroupIds := make([]*string, 0, 10)
	for _, entry := range eksCluster.SecurityGroups {
		securityGroupId, err := CreateSecurityGroup(ec2Client, entry, vpcId)
		if err != nil {
			return err
		}
		securityGroupIds = append(securityGroupIds, &securityGroupId)
	}

	describeSubnetsOutput, err := ec2Client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(vpcId),
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to describe subnets: %s", err.Error())
	}

	subnetIds := make([]*string, 0, 10)

	for _, entry := range describeSubnetsOutput.Subnets {
		subnetIds = append(subnetIds, entry.SubnetId)
		log.Printf("Find subnet %s", *(entry.SubnetId))
	}

	if len(subnetIds) == 0 {
		return fmt.Errorf("did not find subnet in describeSubnetsOutput: %v", describeSubnetsOutput)
	}

	iamClient := iam.New(session)

	amazonEKSClusterPolicyName := "AmazonEKSClusterPolicy"
	amazonEKSClusterPolicy, err := awslib.FindIamPolicy(iamClient, amazonEKSClusterPolicyName)
	if err != nil {
		return fmt.Errorf("failed to find IAM policy %s: %s", amazonEKSClusterPolicyName, err.Error())
	}

	if amazonEKSClusterPolicy == nil {
		return fmt.Errorf("did not find IAM policy %s", amazonEKSClusterPolicyName)
	}

	assumeRolePolicyDocument := eksCluster.ControlPlaneRole.AssumeRolePolicyDocument
	roleName := eksCluster.ControlPlaneRole.Name
	createRoleResult, err := iamClient.CreateRole(&iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(assumeRolePolicyDocument),
	})
	if err != nil {
		if !awslib.AlreadyExistsMessage(err.Error()) {
			return fmt.Errorf("failed to create IAM role %s with EKS cluster policy: %s", roleName, err.Error())
		} else {
			log.Printf("Find and use existing IAM role to attach EKS cluster policy: %s", roleName)
		}
	} else {
		log.Printf("Created IAM role %s to attach EKS cluster policy", *createRoleResult.Role.RoleName)
	}

	for _, entry := range eksCluster.ControlPlaneRole.ExtraPolicyArns {
		err = awslib.AttachToRoleByPolicyArn(iamClient, roleName, entry)
		if err != nil {
			return fmt.Errorf("failed to attach IAM policy %s to role %s: %s", entry, roleName, err.Error())
		}
	}

	getRoleOutput, err := iamClient.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return fmt.Errorf("failed to get role %s: %s", roleName, err.Error())
	}

	createClusterInput := eks.CreateClusterInput{
		Name: aws.String(clusterName),
		ResourcesVpcConfig: &eks.VpcConfigRequest{
			SecurityGroupIds: securityGroupIds,
			SubnetIds: subnetIds,
			// TODO make following configurable
			EndpointPrivateAccess: aws.Bool(true),
			EndpointPublicAccess:  aws.Bool(true),
		},
		RoleArn: getRoleOutput.Role.Arn,
		Version: aws.String("1.21"),
	}

	eksClient := eks.New(session)

	createEksClusterErr := awslib.CreateEksCluster(eksClient, &createClusterInput)
	if createEksClusterErr != nil {
		if !awslib.AlreadyExistsMessage(createEksClusterErr.Error()) {
			return fmt.Errorf("failed to create EKS cluster %s: %s", clusterName, createEksClusterErr.Error())
		} else {
			log.Printf("EKS cluster %s already exists, do not create it again", clusterName)
		}
	}

	waitClusterReadyErr := common.RetryUntilTrue(func() (bool, error) {
		targetClusterStatus := "ACTIVE"
		describeClusterOutput, err := eksClient.DescribeCluster(&eks.DescribeClusterInput{
			Name: aws.String(clusterName),
		})
		if err != nil {
			return false, err
		}
		clusterStatus := *describeClusterOutput.Cluster.Status
		ready := strings.EqualFold(clusterStatus, targetClusterStatus)
		if !ready {
			log.Printf("EKS cluster %s not ready (in status: %s), waiting... (EKS may take 10+ minutes to be ready, thanks for your patience)", clusterName, clusterStatus)
		} else {
			log.Printf("EKS cluster %s is ready (in status: %s)", clusterName, clusterStatus)
		}
		return ready, nil
	}, 30*time.Minute, 30*time.Second)

	if waitClusterReadyErr != nil {
		return fmt.Errorf("failed to wait ready for cluster %s: %s", clusterName, waitClusterReadyErr.Error())
	}

	return nil
}