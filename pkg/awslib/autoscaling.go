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
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/eks"
)

func CreateOrUpdateClusterAutoscalerTagsOnNodeGroup(region string, eksClusterName string, nodeGroupName string) error {
	session := CreateSession(region)

	eksClient := eks.New(session)
	describeNodegroupOutput, err := eksClient.DescribeNodegroup(&eks.DescribeNodegroupInput{
		ClusterName: &eksClusterName,
		NodegroupName: &nodeGroupName,
	})
	if err != nil {
		return fmt.Errorf("failed to describe node group %s in EKS cluster %s: %s", nodeGroupName, eksClusterName, err.Error())
	}

	autoScalingGroups := describeNodegroupOutput.Nodegroup.Resources.AutoScalingGroups
	if len(autoScalingGroups) != 1 {
		return fmt.Errorf("invalid result, describe node group %s in EKS cluster %s: AutoScalingGroups has size %d (expected 1)", nodeGroupName, eksClusterName, len(autoScalingGroups))
	}

	resourceId := autoScalingGroups[0].Name

	svc := autoscaling.New(session)
	key1 := fmt.Sprintf("k8s.io/cluster-autoscaler/%s", eksClusterName)
	_, err = svc.CreateOrUpdateTags(&autoscaling.CreateOrUpdateTagsInput{
		Tags: []*autoscaling.Tag{
			&autoscaling.Tag{
				ResourceId: resourceId,
				ResourceType: aws.String("auto-scaling-group"),
				Key: aws.String(key1),
				Value: aws.String("owned"),
				PropagateAtLaunch: aws.Bool(true),
			},
			&autoscaling.Tag{
				ResourceId: resourceId,
				ResourceType: aws.String("auto-scaling-group"),
				Key: aws.String("k8s.io/cluster-autoscaler/enabled"),
				Value: aws.String("TRUE"),
				PropagateAtLaunch: aws.Bool(true),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create or update tags for: %s", err.Error())
	}
	return nil
}

