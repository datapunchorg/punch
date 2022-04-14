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
	"github.com/aws/aws-sdk-go/service/kafka"
)

func GetKafkaClusterInfo(region string, clusterName string) (kafka.ClusterInfo, error) {
	var result kafka.ClusterInfo
	session := CreateSession(region)
	// see https://github.com/aws/aws-sdk-go/blob/main/service/kafka/service.go
	svc := kafka.New(session)
	listClustersOutput, err := svc.ListClusters(&kafka.ListClustersInput{
		ClusterNameFilter: &clusterName,
	})
	if err != nil {
		return result, fmt.Errorf("failed to list clusters: %s", err.Error())
	}
	if len(listClustersOutput.ClusterInfoList) == 0 {
		return result, fmt.Errorf("got empty result when list clusters by name: %s", clusterName)
	}
	if len(listClustersOutput.ClusterInfoList) > 1 {
		return result, fmt.Errorf("got multiple clusters when list clusters by name: %s", clusterName)
	}
	result = *listClustersOutput.ClusterInfoList[0]
	return result, nil
}
