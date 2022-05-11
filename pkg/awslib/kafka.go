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

package awslib

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/kafka"
)

func CheckKafkaCluster(region string, clusterName string) (*kafka.ClusterInfo, error) {
	session := CreateSession(region)
	svc := kafka.New(session)
	listClustersOutput, err := svc.ListClusters(&kafka.ListClustersInput{
		ClusterNameFilter: &clusterName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %s", err.Error())
	}
	if len(listClustersOutput.ClusterInfoList) == 0 {
		return nil, nil
	}
	if len(listClustersOutput.ClusterInfoList) > 1 {
		return nil, fmt.Errorf("got multiple clusters when list clusters by name: %s", clusterName)
	}
	result := listClustersOutput.ClusterInfoList[0]
	return result, nil
}

func GetKafkaClusterInfo(region string, clusterName string) (kafka.ClusterInfo, error) {
	checkResult, err := CheckKafkaCluster(region, clusterName)
	if err != nil {
		return kafka.ClusterInfo{}, fmt.Errorf("failed to get Kafka cluster %s: %s", clusterName, err.Error())
	}
	if checkResult == nil {
		return kafka.ClusterInfo{}, fmt.Errorf("did not get Kafka cluster %s: %s", clusterName, err.Error())
	}
	return *checkResult, nil
}
