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
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/common"
	"log"
	"strings"
	"time"
)

type NodeGroup struct {
	Name          string   `json:"name" yaml:"name"`
	InstanceTypes []string `json:"instanceTypes" yaml:"instanceTypes"`
	DesiredSize   int64
	MaxSize       int64
	MinSize       int64
}

func CreateNodeGroup(region string, clusterName string, nodeGroup NodeGroup, nodeRoleArn string) error {
	log.Printf("Adding node group to EKS cluster %s in AWS region: %v", clusterName, region)

	_, eksClient := awslib.GetEksClient(region)

	describeClusterOutput, err := eksClient.DescribeCluster(&eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})

	if err != nil {
		return fmt.Errorf("failed to get EKS cluster %s: %s", clusterName, err.Error())
	}

	cluster := describeClusterOutput.Cluster

	_, createNodeGroupErr := eksClient.CreateNodegroup(&eks.CreateNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: aws.String(nodeGroup.Name),
		InstanceTypes: aws.StringSlice(nodeGroup.InstanceTypes),
		ScalingConfig: &eks.NodegroupScalingConfig{
			DesiredSize: aws.Int64(int64(nodeGroup.DesiredSize)),
			MaxSize:     aws.Int64(int64(nodeGroup.MaxSize)),
			MinSize:     aws.Int64(int64(nodeGroup.MinSize)),
		},
		Subnets:  cluster.ResourcesVpcConfig.SubnetIds,
		NodeRole: aws.String(nodeRoleArn),
	})

	if createNodeGroupErr != nil {
		if !awslib.AlreadyExistsMessage(createNodeGroupErr.Error()) {
			return fmt.Errorf("failed to create node group %s in EKS cluster %s: %s", nodeGroup.Name, clusterName, createNodeGroupErr.Error())
		} else {
			log.Printf("Node group %s already in EKS cluster %s, do not add the node group again", nodeGroup.Name, clusterName)
		}
	} else {
		log.Printf("Added node group %s in EKS cluster %s", nodeGroup.Name, clusterName)
	}

	waitReadyErr := common.RetryUntilTrue(func() (bool, error) {
		targetStatus := "ACTIVE"
		describeOutput, err := eksClient.DescribeNodegroup(&eks.DescribeNodegroupInput{
			ClusterName:   aws.String(clusterName),
			NodegroupName: aws.String(nodeGroup.Name),
		})
		if err != nil {
			return false, err
		}
		status := *describeOutput.Nodegroup.Status
		ready := strings.EqualFold(status, targetStatus)
		if !ready {
			log.Printf("Node group %s not ready (in status: %s), waiting...", nodeGroup.Name, status)
		} else {
			log.Printf("Node group %s is ready (in status: %s)", nodeGroup.Name, status)
		}
		return ready, nil
	}, 30*time.Minute, 30*time.Second)

	if waitReadyErr != nil {
		return fmt.Errorf("failed to wait ready for node group %s: %s", nodeGroup.Name, err.Error())
	}

	return nil
}
