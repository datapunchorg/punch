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
	"github.com/aws/aws-sdk-go/service/s3"
	"log"
)

func CreateS3Bucket(region string, bucketName string) error {
	session := CreateSession(region)

	log.Printf("Creating S3 bucket in AWS region: %v", region)

	s3Client := s3.New(session)

	listBucketsOutput, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("failed to list buckets: %s", err.Error())
	}

	var bucket *s3.Bucket
	for _, entry := range listBucketsOutput.Buckets {
		if *entry.Name == bucketName {
			bucket = entry
			log.Printf("Bucket %s already exists, do not create it again", bucketName)
			break
		}
	}

	if bucket == nil {
		_, err := s3Client.CreateBucket(&s3.CreateBucketInput{
			Bucket: aws.String(bucketName),
			CreateBucketConfiguration: &s3.CreateBucketConfiguration{
				LocationConstraint: aws.String(region),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create bucket %s: %s", bucketName, err.Error())
		} else {
			log.Printf("Created bucket %s", bucketName)
		}
	}

	return nil
}
