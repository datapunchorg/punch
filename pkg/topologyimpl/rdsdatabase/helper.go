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

package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/common"
	"github.com/datapunchorg/punch/pkg/resource"
)

func CreateDatabase(databaseSpec RdsDatabaseTopologySpec) (*rds.DBCluster, error) {
	databaseId := databaseSpec.DatabaseId
	session := awslib.CreateSession(databaseSpec.Region)

	ec2Client := ec2.New(session)
	securityGroupIds := make([]*string, 0, 10)
	for _, entry := range databaseSpec.SecurityGroups {
		securityGroupId, err := resource.CreateSecurityGroup(ec2Client, databaseSpec.VpcId, entry)
		if err != nil {
			return nil, err
		}
		securityGroupIds = append(securityGroupIds, &securityGroupId)
	}

	svc := rds.New(session)
	input := &rds.CreateDBClusterInput{
		AvailabilityZones:     aws.StringSlice(databaseSpec.AvailabilityZones),
		BackupRetentionPeriod: aws.Int64(1),
		DBClusterIdentifier:   aws.String(strings.ToLower(databaseId)),
		// DBClusterParameterGroupName: aws.String("parametergroup-01"),
		DatabaseName:        aws.String(normalizeDatabaseName(databaseId)),
		Engine:              aws.String(databaseSpec.Engine),
		EngineVersion:       aws.String(databaseSpec.EngineVersion),
		EngineMode:          aws.String(databaseSpec.EngineMode),
		MasterUsername:      aws.String(databaseSpec.MasterUserName),
		MasterUserPassword:  aws.String(databaseSpec.MasterUserPassword),
		StorageEncrypted:    aws.Bool(true),
		EnableHttpEndpoint:  aws.Bool(true),
		VpcSecurityGroupIds: securityGroupIds,
	}
	if databaseSpec.Port != 0 {
		input.Port = aws.Int64(databaseSpec.Port)
	}
	result, err := svc.CreateDBCluster(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case rds.ErrCodeDBClusterAlreadyExistsFault:
				log.Printf("Database %s already exists, do not create it again", databaseId)
			case rds.ErrCodeInsufficientStorageClusterCapacityFault:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeInsufficientStorageClusterCapacityFault, aerr.Error())
			case rds.ErrCodeDBClusterQuotaExceededFault:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeDBClusterQuotaExceededFault, aerr.Error())
			case rds.ErrCodeStorageQuotaExceededFault:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeStorageQuotaExceededFault, aerr.Error())
			case rds.ErrCodeDBSubnetGroupNotFoundFault:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeDBSubnetGroupNotFoundFault, aerr.Error())
			case rds.ErrCodeInvalidVPCNetworkStateFault:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeInvalidVPCNetworkStateFault, aerr.Error())
			case rds.ErrCodeInvalidDBClusterStateFault:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeInvalidDBClusterStateFault, aerr.Error())
			case rds.ErrCodeInvalidDBSubnetGroupStateFault:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeInvalidDBSubnetGroupStateFault, aerr.Error())
			case rds.ErrCodeInvalidSubnet:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeInvalidSubnet, aerr.Error())
			case rds.ErrCodeInvalidDBInstanceStateFault:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeInvalidDBInstanceStateFault, aerr.Error())
			case rds.ErrCodeDBClusterParameterGroupNotFoundFault:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeDBClusterParameterGroupNotFoundFault, aerr.Error())
			case rds.ErrCodeKMSKeyNotAccessibleFault:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeKMSKeyNotAccessibleFault, aerr.Error())
			case rds.ErrCodeDBClusterNotFoundFault:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeDBClusterNotFoundFault, aerr.Error())
			case rds.ErrCodeDBInstanceNotFoundFault:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeDBInstanceNotFoundFault, aerr.Error())
			case rds.ErrCodeDBSubnetGroupDoesNotCoverEnoughAZs:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeDBSubnetGroupDoesNotCoverEnoughAZs, aerr.Error())
			case rds.ErrCodeGlobalClusterNotFoundFault:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeGlobalClusterNotFoundFault, aerr.Error())
			case rds.ErrCodeInvalidGlobalClusterStateFault:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeInvalidGlobalClusterStateFault, aerr.Error())
			case rds.ErrCodeDomainNotFoundFault:
				return nil, fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeDomainNotFoundFault, aerr.Error())
			default:
				return nil, fmt.Errorf("failed to create database %s: %s", databaseId, aerr.Error())
			}
		} else {
			return nil, fmt.Errorf("failed to create database %s: %s", databaseId, err.Error())
		}
	} else {
		log.Printf("Created database: %v", result)
	}
	var dbCluster *rds.DBCluster = nil
	err = common.RetryUntilTrue(func() (bool, error) {
		describeDBClustersOutput, err := svc.DescribeDBClusters(&rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(databaseId),
		})
		if err != nil {
			return false, fmt.Errorf("failed to get DB cluster %s: %s", databaseId, err.Error())
		}
		if len(describeDBClustersOutput.DBClusters) == 0 {
			return false, fmt.Errorf("failed to get database %s: got empty result when listing by database id", databaseId)
		}
		if len(describeDBClustersOutput.DBClusters) > 1 {
			return false, fmt.Errorf("failed to get database %s: got %d databases when listing by database id", databaseId, len(describeDBClustersOutput.DBClusters))
		}
		if describeDBClustersOutput.DBClusters[0] == nil {
			return false, fmt.Errorf("failed to get database %s: got nil pointer when listing by database id", databaseId)
		}
		dbCluster = describeDBClustersOutput.DBClusters[0]
		if strings.EqualFold(*dbCluster.Status, "available") {
			log.Printf("Database %s is ready (current status: %s)", databaseId, *dbCluster.Status)
			return true, nil
		} else {
			log.Printf("Database %s not ready (current status: %s)", databaseId, *dbCluster.Status)
			return false, nil
		}
	},
		20*time.Minute,
		10*time.Second)

	if err != nil {
		return nil, fmt.Errorf("failed to get database %s: %s", databaseId, err.Error())
	}
	if dbCluster == nil {
		return nil, fmt.Errorf("failed to get database %s, got null database cluster", databaseId)
	}
	return dbCluster, nil
}

func DeleteDatabase(region string, databaseId string) error {
	session := awslib.CreateSession(region)
	svc := rds.New(session)
	input := &rds.DeleteDBClusterInput{
		DBClusterIdentifier: aws.String(databaseId),
		SkipFinalSnapshot:   aws.Bool(true),
	}
	_, err := svc.DeleteDBCluster(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case rds.ErrCodeDBClusterNotFoundFault:
				log.Printf("Database %s does not exist in region %s, do not delete it", databaseId, region)
				return nil
			case rds.ErrCodeInvalidDBClusterStateFault:
				return fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeInvalidDBClusterStateFault, aerr.Error())
			case rds.ErrCodeDBClusterSnapshotAlreadyExistsFault:
				return fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeDBClusterSnapshotAlreadyExistsFault, aerr.Error())
			case rds.ErrCodeSnapshotQuotaExceededFault:
				return fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeSnapshotQuotaExceededFault, aerr.Error())
			case rds.ErrCodeInvalidDBClusterSnapshotStateFault:
				return fmt.Errorf("failed to create database %s: %s %s", databaseId, rds.ErrCodeInvalidDBClusterSnapshotStateFault, aerr.Error())
			default:
				return aerr
			}
		} else {
			return err
		}
	}

	log.Printf("Sent request to delete database %s in region %s", databaseId, region)

	err = common.RetryUntilTrue(func() (bool, error) {
		describeDBClustersOutput, err := svc.DescribeDBClusters(&rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(databaseId),
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == rds.ErrCodeDBClusterNotFoundFault {
					log.Printf("Database %s does not exist in region %s, finish deleting", databaseId, region)
					return true, nil
				} else {
					return false, fmt.Errorf("failed to get DB cluster %s: %s", databaseId, err.Error())
				}
			}
		}
		if len(describeDBClustersOutput.DBClusters) == 0 {
			log.Printf("Database %s does not exist in region %s, finish deleting", databaseId, region)
			return true, nil
		}
		if len(describeDBClustersOutput.DBClusters) > 1 {
			return false, fmt.Errorf("failed to get database %s: got %d databases when listing by database id", databaseId, len(describeDBClustersOutput.DBClusters))
		}
		if describeDBClustersOutput.DBClusters[0] == nil {
			return false, fmt.Errorf("failed to get database %s: got nil pointer when listing by database id", databaseId)
		}
		log.Printf("Database %s still exists in region %s, waiting", databaseId, region)
		return false, nil
	},
		20*time.Minute,
		10*time.Second)

	if err != nil {
		return fmt.Errorf("failed to check database %s: %s", databaseId, err.Error())
	}
	return nil
}
