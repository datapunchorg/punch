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

package resource

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/common"
	"github.com/datapunchorg/punch/pkg/framework"
	"log"
	"strings"
	"time"
)

type EKSCluster struct {
	ClusterName string `json:"clusterName" yaml:"clusterName"`
	SubnetIds         []string                  `json:"subnetIds" yaml:"subnetIds"`
	// The Amazon Resource Name (ARN) of the IAM role that provides permissions
	// for the Kubernetes control plane to make calls to Amazon Web Services API
	// operations on your behalf. For more information, see Amazon EKS Service IAM
	// Role (https://docs.aws.amazon.com/eks/latest/userguide/service_IAM_role.html)
	// in the Amazon EKS User Guide.
	ControlPlaneRole IamRole `json:"controlPlaneRole" yaml:"controlPlaneRole"`
	// See https://docs.aws.amazon.com/eks/latest/userguide/sec-group-reqs.html
	SecurityGroups []SecurityGroup `json:"securityGroups" yaml:"securityGroups"`
	InstanceRole   IamRole         `json:"instanceRole" yaml:"instanceRole"`
}

type EKSClusterSummary struct {
	ClusterName string `json:"clusterName" yaml:"clusterName"`
	OidcIssuer  string `json:"oidcIssuer" yaml:"oidcIssuer"`
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

	if err != nil {
		return err
	}

	return nil
}

func DescribeEksCluster(region string, clusterName string) (EKSClusterSummary, error) {
	session := awslib.CreateSession(region)
	eksClient := eks.New(session)

	describeClusterOutput, err := eksClient.DescribeCluster(&eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})

	if err != nil {
		return EKSClusterSummary{}, fmt.Errorf("failed to describe EKS cluster %s, %s", clusterName, err.Error())
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

func CheckEksCluster(region string, clusterName string) error {
	session := awslib.CreateSession(region)
	eksClient := eks.New(session)

	_, err := eksClient.DescribeCluster(&eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})

	if err != nil {
		return fmt.Errorf("failed to get EKS cluster %s, %s", clusterName, err.Error())
	}
	return nil
}

func GetEksNginxLoadBalancerUrls(commandEnvironment framework.CommandEnvironment, region string, eksClusterName string, nginxNamespace string, nginxServiceName string, explicitHttpsPort int32) ([]string, error) {
	hostPorts, err := awslib.GetLoadBalancerHostPorts(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName, nginxNamespace, nginxServiceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get load balancer urls for nginx controller service %s in namespace %s: %s", nginxServiceName, nginxNamespace, err.Error())
	}

	dnsNamesMap := make(map[string]bool, len(hostPorts))
	for _, entry := range hostPorts {
		dnsNamesMap[entry.Host] = true
	}
	dnsNames := make([]string, 0, len(dnsNamesMap))
	for k := range dnsNamesMap {
		dnsNames = append(dnsNames, k)
	}

	if !commandEnvironment.GetBoolOrElse(framework.CmdEnvWithMinikube, false) {
		err = awslib.WaitLoadBalancersReadyByDnsNames(region, dnsNames)
		if err != nil {
			return nil, fmt.Errorf("failed to wait and get load balancer urls: %s", err.Error())
		}
	}

	urls := make([]string, 0, len(hostPorts))
	for _, entry := range hostPorts {
		if entry.Port == 443 {
			urls = append(urls, fmt.Sprintf("https://%s", entry.Host))
		} else if entry.Port == explicitHttpsPort {
			urls = append(urls, fmt.Sprintf("https://%s:%d", entry.Host, entry.Port))
		} else if entry.Port == 80 {
			urls = append(urls, fmt.Sprintf("http://%s", entry.Host))
		} else {
			urls = append(urls, fmt.Sprintf("http://%s:%d", entry.Host, entry.Port))
		}
	}

	return urls, nil
}

func GetLoadBalancerPreferredUrl(urls []string) string {
	preferredUrl := ""
	if len(urls) > 0 {
		preferredUrl = urls[0]
		for _, entry := range urls {
			// Use https url if possible
			if strings.HasPrefix(strings.ToLower(entry), "https") {
				preferredUrl = entry
			}
		}
	}
	return preferredUrl
}
