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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
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

func UploadDataToS3Url(region string, url string, data io.Reader) error {
	bucket, key, err := GetS3BucketAndKeyFromUrl(url)
	if err != nil {
		return err
	}
	return UploadDataToS3(region, bucket, key, data)
}

func UploadDataToS3(region string, bucket string, key string, data io.Reader) error {
	session, err := CreateSessionOrError(region)
	if err != nil {
		return err
	}

	// See http://docs.aws.amazon.com/sdk-for-go/api/service/s3/s3manager/#NewUploader
	uploader := s3manager.NewUploader(session)

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key: aws.String(key),
		Body: data,
	})
	if err != nil {
		return fmt.Errorf("failed to upload %s to S3 bucket %s: %s", key, bucket, err.Error())
	}

	log.Printf("Uploaded %s to S3 bucket %s", key, bucket)
	return nil
}
