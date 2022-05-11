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
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/datapunchorg/punch/pkg/common"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"github.com/google/uuid"
	"io/fs"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
	"time"
)

func CreateEksCluster(eksClient *eks.EKS, createClusterInput *eks.CreateClusterInput) error {
	createClusterInputCopy := *createClusterInput

	if createClusterInputCopy.Name == nil || *createClusterInputCopy.Name == "" {
		return fmt.Errorf("cluster name is empty")
	}

	name := *createClusterInputCopy.Name
	if createClusterInputCopy.ClientRequestToken == nil {
		createClusterInputCopy.ClientRequestToken = aws.String(fmt.Sprintf("create-eks-%s-%v", name, uuid.New().String()))
	}

	log.Printf("Creating EKS cluster: %v", createClusterInputCopy)

	createClusterOutput, err := eksClient.CreateCluster(&createClusterInputCopy)
	if err != nil {
		return err
	}

	log.Printf("Created EKS cluster: %v", createClusterOutput)
	return nil
}

func ListEksClusters(eksClient *eks.EKS) ([]string, error) {
	result := make([]string, 0, 100)
	var nextToken *string
	var hasMoreResult = true
	for hasMoreResult {
		listClustersOutput, err := eksClient.ListClusters(&eks.ListClustersInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, entry := range listClustersOutput.Clusters {
			result = append(result, *entry)
		}
		nextToken = listClustersOutput.NextToken
		hasMoreResult = nextToken != nil
	}
	return result, nil
}

func NodeGroupExists(eksClient *eks.EKS, clusterName string, nodeGroupName string) (bool, error) {
	listOutput, err := eksClient.ListNodegroups(&eks.ListNodegroupsInput{
		ClusterName: aws.String(clusterName),
	})
	if err != nil {
		return false, err
	}
	exists := false
	for _, entry := range listOutput.Nodegroups {
		if *entry == nodeGroupName {
			exists = true
			break
		}
	}
	return exists, nil
}

func GetEksClient(region string) (*session.Session, *eks.EKS) {
	session := CreateSession(region)

	eksClient := eks.New(session)
	return session, eksClient
}

func CreateEksKubeConfig(region string, clusterName string) (kubelib.KubeConfig, error) {
	session, eksClient := GetEksClient(region)

	describeClusterOutput, err := eksClient.DescribeCluster(&eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})

	if err != nil {
		return kubelib.KubeConfig{}, err
	}

	log.Printf("Getting client for EKS Cluster: %s", clusterName)

	cluster := describeClusterOutput.Cluster

	tokenGenerator, err := token.NewGenerator(true, true)
	if err != nil {
		return kubelib.KubeConfig{}, err
	}

	token, err := tokenGenerator.GetWithOptions(&token.GetTokenOptions{
		Session:   session,
		ClusterID: aws.StringValue(cluster.Name),
		// AssumeRoleARN: "arn:aws:iam::123456789:role/MyRoleName",
	})
	if err != nil {
		return kubelib.KubeConfig{}, err
	}

	ca, err := base64.StdEncoding.DecodeString(aws.StringValue(cluster.CertificateAuthority.Data))
	if err != nil {
		return kubelib.KubeConfig{}, err
	}

	caFile, err := os.CreateTemp("", "ca.txt")
	if err != nil {
		caFile.Close()
		return kubelib.KubeConfig{}, err
	}
	caFile.Close()
	ioutil.WriteFile(caFile.Name(), ca, fs.ModePerm)

	kubeConfig := kubelib.KubeConfig{
		ApiServer: *cluster.Endpoint,
		CAFile:    caFile.Name(),
		CA:        ca,
		KubeToken: token.Token,
	}

	return kubeConfig, nil
}

func CreateKubeConfig(region string, kubeConfigFile string, clusterName string) (kubelib.KubeConfig, error) {
	var kubeConfig kubelib.KubeConfig
	if kubeConfigFile != "" {
		kubeConfig = kubelib.KubeConfig{
			ConfigFile: kubeConfigFile,
		}
	} else {
		var err error
		kubeConfig, err = CreateEksKubeConfig(region, clusterName)
		if err != nil {
			return kubelib.KubeConfig{}, err
		}
	}
	return kubeConfig, nil
}

func CreateEksKubernetesClient(region string, clusterName string) (kubelib.KubeConfig, *kubernetes.Clientset, error) {
	kubeConfig, err := CreateEksKubeConfig(region, clusterName)
	if err != nil {
		return kubeConfig, nil, err
	}

	config := rest.Config{
		Host:            kubeConfig.ApiServer,
		BearerToken:     kubeConfig.KubeToken,
		TLSClientConfig: rest.TLSClientConfig{CAData: kubeConfig.CA},
	}

	clientset, err := kubernetes.NewForConfig(&config)
	if err != nil {
		return kubelib.KubeConfig{}, nil, fmt.Errorf("failed to create Kubernetes client for cluster %s (%s): %s", clusterName, kubeConfig.ApiServer, err.Error())
	}
	return kubeConfig, clientset, nil
}

func CreateKubernetesClient(region string, kubeConfigFile string, clusterName string) (kubelib.KubeConfig, *kubernetes.Clientset, error) {
	var kubeConfig kubelib.KubeConfig
	var clientset *kubernetes.Clientset
	if kubeConfigFile != "" {
		kubeConfig = kubelib.KubeConfig{
			ConfigFile: kubeConfigFile,
		}
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			return kubelib.KubeConfig{}, nil, err
		}
		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			return kubelib.KubeConfig{}, nil, err
		}
	} else {
		var err error
		kubeConfig, clientset, err = CreateEksKubernetesClient(region, clusterName)
		if err != nil {
			return kubelib.KubeConfig{}, nil, err
		}
	}
	return kubeConfig, clientset, nil
}

func DeleteEKSCluster(region string, clusterName string) error {
	session := CreateSession(region)

	log.Printf("Deleting EKS cluster %s in AWS region: %v", clusterName, region)

	eksClient := eks.New(session)

	_, err := eksClient.DeleteCluster(&eks.DeleteClusterInput{
		Name: aws.String(clusterName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete EKS cluster %s: %s", clusterName, err.Error())
	}

	waitDeletedErr := common.RetryUntilTrue(func() (bool, error) {
		listOutput, err := ListEksClusters(eksClient)
		if err != nil {
			return false, err
		}
		stillExists := false
		for _, entry := range listOutput {
			if entry == clusterName {
				stillExists = true
				break
			}
		}
		if stillExists {
			log.Printf("Cluster %s still exists, waiting...", clusterName)
		} else {
			log.Printf("Cluster %s does not exist", clusterName)
		}
		return !stillExists, nil
	}, 10*time.Minute, 30*time.Second)

	if waitDeletedErr != nil {
		return fmt.Errorf("failed to wait cluster %s to be deleted: %s", clusterName, waitDeletedErr.Error())
	}

	return nil
}
