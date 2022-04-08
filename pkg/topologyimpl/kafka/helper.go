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

package kafka

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kafka"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/resource"
	"log"
)

func CreateKafkaCluster(spec KafkaTopologySpec) (string, error) {
	session := awslib.CreateSession(spec.Region)
	clusterName := spec.ClusterName
	// see https://github.com/aws/aws-sdk-go/blob/main/service/kafka/service.go
	svc := kafka.New(session)
	subnetIds, err := resource.GetSubnetIds(spec.Region, spec.VpcId)
	if err != nil {
		return "", err
	}
	// TODO
	// Specify exactly two subnets if you are using the US West (N. California) Region. For other Regions where Amazon MSK is available, you can specify either two or three subnets. The subnets that you specify must be in distinct Availability Zones. When you create a cluster, Amazon MSK distributes the broker nodes evenly across the subnets that you specify.
	// Client subnets can't be in Availability Zone us-east-1e.
	createResult, err := svc.CreateClusterV2(&kafka.CreateClusterV2Input{
		ClusterName: &clusterName,
		Provisioned: &kafka.ProvisionedRequest{
			BrokerNodeGroupInfo: &kafka.BrokerNodeGroupInfo{
				ClientSubnets: aws.StringSlice(subnetIds),
				InstanceType: aws.String(DefaultInstanceType),
			},
			KafkaVersion: aws.String("2.8.1"),
			NumberOfBrokerNodes: aws.Int64(int64(len(subnetIds))),
		},
		/* Serverless: &kafka.ServerlessRequest{
			ClientAuthentication: &kafka.ServerlessClientAuthentication{
				Sasl: &kafka.ServerlessSasl{
					Iam: &kafka.Iam{
						Enabled: aws.Bool(true),
					},
				},
			},
			VpcConfigs: []*kafka.VpcConfig{
				&kafka.VpcConfig{
					SecurityGroupIds: []*string{},
					SubnetIds: []*string{},
				},
			},
		}, */
	})

	if err != nil {
		if awslib.AlreadyExistsMessage(err.Error()) {
			log.Printf("Kafka cluster %s already exists, do not create it again", clusterName)
		} else {
			return "", fmt.Errorf("failed to create Kafka cluster %s: %s", clusterName, err.Error())
		}
	} else {
		log.Printf("Created Kafka cluster %s: %v", clusterName, createResult)
	}

	// TODO wait cluster ready and return cluster information
	return "", nil
}
