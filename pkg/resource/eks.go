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
	SubnetIds         []string                  `json:"subnetIds" yaml:"subnetIds"`
	// The Amazon Resource Name (ARN) of the IAM role that provides permissions
	// for the Kubernetes control plane to make calls to Amazon Web Services API
	// operations on your behalf. For more information, see Amazon Eks Service IAM
	// Role (https://docs.aws.amazon.com/eks/latest/userguide/service_IAM_role.html)
	// in the Amazon Eks User Guide.
	ControlPlaneRole IAMRole `json:"controlPlaneRole" yaml:"controlPlaneRole"`
	// See https://docs.aws.amazon.com/eks/latest/userguide/sec-group-reqs.html
	SecurityGroups []SecurityGroup `json:"securityGroups" yaml:"securityGroups"`
	InstanceRole   IAMRole         `json:"instanceRole" yaml:"instanceRole"`
}

type EKSClusterSummary struct {
	ClusterName string `json:"clusterName" yaml:"clusterName"`
	OidcIssuer  string `json:"oidcIssuer" yaml:"oidcIssuer"`
}

func CreateEksCluster(region string, vpcId string, eksCluster EKSCluster) error {
	clusterName := eksCluster.ClusterName

	session := awslib.CreateSession(region)

	log.Printf("Creating Eks cluster in AWS region: %v", region)

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

	securityGroupIds, err := CreateSecurityGroups(region, vpcId, eksCluster.SecurityGroups)
	if err != nil {
		return fmt.Errorf("failed to create security groups in region %s vpc %s: %s", region, vpcId, err.Error())
	}

	subnetIds := eksCluster.SubnetIds
	if len(subnetIds) == 0 {
		subnetIds, err = GetSubnetIds(region, vpcId)
		if err != nil {
			return fmt.Errorf("failed to get subnect ids for region %s vpc %s: %s", region, vpcId, err.Error())
		}
		if len(subnetIds) == 0 {
			return fmt.Errorf("did not get any subnect id for region %s vpc %s: %s", region, vpcId, err.Error())
		}
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
			return fmt.Errorf("failed to create IAM role %s with Eks cluster policy: %s", roleName, err.Error())
		} else {
			log.Printf("Find and use existing IAM role to attach Eks cluster policy: %s", roleName)
		}
	} else {
		log.Printf("Created IAM role %s to attach Eks cluster policy", *createRoleResult.Role.RoleName)
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
			SecurityGroupIds: aws.StringSlice(securityGroupIds),
			SubnetIds:        aws.StringSlice(subnetIds),
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
			return fmt.Errorf("failed to create Eks cluster %s: %s", clusterName, createEksClusterErr.Error())
		} else {
			log.Printf("Eks cluster %s already exists, do not create it again", clusterName)
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
			log.Printf("Eks cluster %s not ready (in status: %s), waiting... (Eks may take 10+ minutes to be ready, thanks for your patience)", clusterName, clusterStatus)
		} else {
			log.Printf("Eks cluster %s is ready (in status: %s)", clusterName, clusterStatus)
		}
		return ready, nil
	}, 30*time.Minute, 30*time.Second)

	if waitClusterReadyErr != nil {
		return fmt.Errorf("failed to wait ready for cluster %s: %s", clusterName, waitClusterReadyErr.Error())
	}

	if err != nil {
		return err
	}

	return nil
}

func DescribeEksCluster(region string, clusterName string) (EKSClusterSummary, error) {
	session := awslib.CreateSession(region)

	log.Printf("Creating Eks cluster in AWS region: %v", region)

	eksClient := eks.New(session)

	describeClusterOutput, err := eksClient.DescribeCluster(&eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})

	if err != nil {
		return EKSClusterSummary{}, fmt.Errorf("failed to describe Eks cluster %s, %s", clusterName, err.Error())
	}

	oidcIssuer := ""
	if describeClusterOutput.Cluster != nil &&
		describeClusterOutput.Cluster.Identity != nil &&
		describeClusterOutput.Cluster.Identity.Oidc != nil &&
		describeClusterOutput.Cluster.Identity.Oidc.Issuer != nil {
		oidcIssuer = *describeClusterOutput.Cluster.Identity.Oidc.Issuer
	}

	clusterSummary := EKSClusterSummary{
		ClusterName: clusterName,
		OidcIssuer:  oidcIssuer,
	}

	return clusterSummary, nil
}
