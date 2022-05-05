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

package kafkaonmsk

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kafka"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/common"
	"github.com/datapunchorg/punch/pkg/resource"
	"log"
	"strings"
	"time"
)

func CreateKafkaCluster(spec KafkaTopologySpec) (kafka.ClusterInfo, error) {
	var result kafka.ClusterInfo
	session := awslib.CreateSession(spec.Region)
	clusterName := spec.ClusterName
	// see https://github.com/aws/aws-sdk-go/blob/main/service/kafka/service.go
	svc := kafka.New(session)
	subnetIds := spec.SubnetIds
	var err error
	if len(subnetIds) == 0 {
		subnetIds, err = resource.GetSubnetIds(spec.Region, spec.VpcId)
		if err != nil {
			return result, err
		}
		if len(subnetIds) == 0 {
			return result, fmt.Errorf("did not get any subnect id for region %s vpc %s: %s", spec.Region, spec.VpcId, err.Error())
		}
	}
	securityGroupIds, err := resource.CreateSecurityGroups(spec.Region, spec.VpcId, spec.SecurityGroups)
	if err != nil {
		return result, fmt.Errorf("failed to create security groups in region %s vpc %s: %s", spec.Region, spec.VpcId, err.Error())
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
				SecurityGroups: aws.StringSlice(securityGroupIds),
				StorageInfo: &kafka.StorageInfo{
					EbsStorageInfo: &kafka.EBSStorageInfo{
						VolumeSize: aws.Int64(spec.BrokerStorageGB),
					},
				},
			},
			KafkaVersion: &spec.KafkaVersion,
			ClientAuthentication: &kafka.ClientAuthentication{
				// See following for IAM access control on MSK
				// https://docs.aws.amazon.com/msk/latest/developerguide/iam-access-control.html
				// https://github.com/aws/aws-msk-iam-auth
				Sasl: &kafka.Sasl{
					Iam: &kafka.Iam{
						Enabled: aws.Bool(true),
					},
				},
				Tls: &kafka.Tls{
					Enabled: aws.Bool(false),
				},
				Unauthenticated: &kafka.Unauthenticated{
					Enabled: aws.Bool(false),
				},
			},
			NumberOfBrokerNodes: aws.Int64(spec.NumBrokers),
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
			return result, fmt.Errorf("failed to create Kafka cluster %s: %s", clusterName, err.Error())
		}
	} else {
		log.Printf("Created Kafka cluster %s: %v", clusterName, createResult)
	}

	waitClusterReadyErr := common.RetryUntilTrue(func() (bool, error) {
		clusterInfo, err := awslib.GetKafkaClusterInfo(spec.Region, clusterName)
		if err != nil {
			return false, fmt.Errorf("failed to get info for Kafka cluster %s: %s", clusterName, err.Error())
		}
		if strings.EqualFold(*clusterInfo.State, "ACTIVE") {
			log.Printf("Kafka cluster %s is ready in state: %s", clusterName, *clusterInfo.State)
			result = clusterInfo
			return true, nil
		}
		log.Printf("Kafka cluster %s is not ready (current state: %s), waiting for AWS to finish provision... (may take 30+ minutes, thanks for your patience)", clusterName, *clusterInfo.State)
		return false, nil
	}, 60*time.Minute, 30*time.Second)

	if waitClusterReadyErr != nil {
		return kafka.ClusterInfo{}, fmt.Errorf("failed to wait ready for Kafka cluster %s: %s", clusterName, waitClusterReadyErr.Error())
	}

	if result.ClusterArn == nil {
		return kafka.ClusterInfo{}, fmt.Errorf("failed to get information for Kafka cluster %s", clusterName)
	}

	return result, nil
}

func GetBootstrapBrokerString(region string, clusterArn string) (kafka.GetBootstrapBrokersOutput, error) {
	session := awslib.CreateSession(region)
	svc := kafka.New(session)
	getBootstrapBrokersOutput, err := svc.GetBootstrapBrokers(&kafka.GetBootstrapBrokersInput{
		ClusterArn: &clusterArn,
	})
	if err != nil {
		return kafka.GetBootstrapBrokersOutput{}, fmt.Errorf("failed to get bootstrap borkers for cluster %s", clusterArn)
	}
	return *getBootstrapBrokersOutput, nil
}

func DeleteKafkaCluster(region string, clusterName string) error {
	checkResult, err := awslib.CheckKafkaCluster(region, clusterName)
	if err != nil {
		return err
	}
	if checkResult == nil {
		log.Printf("Kafka cluster %s not exist, do not delete it", clusterName)
		return nil
	}
	clusterInfo, err := awslib.GetKafkaClusterInfo(region, clusterName)
	if err != nil {
		return err
	}
	session := awslib.CreateSession(region)
	svc := kafka.New(session)
	_, err = svc.DeleteCluster(&kafka.DeleteClusterInput{
		ClusterArn: clusterInfo.ClusterArn,
	})
	if err != nil {
		return fmt.Errorf("failed to delete Kafka cluster %s: %s", clusterName, err.Error())
	}

	waitClusterDeleteErr := common.RetryUntilTrue(func() (bool, error) {
		checkResult, err := awslib.CheckKafkaCluster(region, clusterName)
		if err != nil {
			return false, fmt.Errorf("failed to check Kafka cluster %s: %s", clusterName, err.Error())
		}
		if checkResult != nil {
			log.Printf("Kafka cluster %s is not deleted (current state: %s), waiting...", clusterName, *checkResult.State)
			return false, nil
		}
		log.Printf("Kafka cluster %s is deleted", clusterName)
		return true, nil
	}, 30*time.Minute, 30*time.Second)

	if waitClusterDeleteErr != nil {
		return fmt.Errorf("failed to wait for Kafka cluster %s to be deleted: %s", clusterName, waitClusterDeleteErr.Error())
	}

	return nil
}