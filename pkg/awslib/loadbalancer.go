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
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/datapunchorg/punch/pkg/common"
	"github.com/datapunchorg/punch/pkg/kubelib"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"strings"
	"time"
)

func DeleteLoadBalancerOnEKS(region string, vpcId string, eksClusterName string, namespace string) error {
	session := CreateSession(region)
	ec2Client := ec2.New(session)

	var err error

	vpcId, err = CheckOrGetFirstVpcId(ec2Client, vpcId)
	if err != nil {
		return fmt.Errorf("failed to check or get first VPC: %s", err.Error())
	}

	_, clientset, err := CreateEksKubernetesClient(region, eksClusterName)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client for cluster %s: %s", eksClusterName, err.Error())
	}

	serviceList, err := clientset.CoreV1().Services(namespace).List(context.TODO(), v12.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list services in namespace %s: %s", namespace, err.Error())
	}
	var ingressHostNames []string
	for _, service := range serviceList.Items {
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			if ingress.Hostname != "" {
				ingressHostNames = append(ingressHostNames, ingress.Hostname)
			}
		}
	}

	if len(ingressHostNames) == 0 {
		log.Printf("Did not find load balancer in namepsace %s, do not delete any load balancer", namespace)
		return nil
	}

	elbClient := elb.New(session)

	loadBalancers, err := ListLoadBalancers(elbClient)
	if err != nil {
		return fmt.Errorf("failed to list load balancers in namespace %s: %s", namespace, err.Error())
	}

	loadBalancersToDelete := make([]*elb.LoadBalancerDescription, 0, 100)
	for _, hostName := range ingressHostNames {
		for _, loadBalancer := range loadBalancers {
			if strings.EqualFold(*loadBalancer.DNSName, hostName) {
				loadBalancersToDelete = append(loadBalancersToDelete, loadBalancer)
			}
		}
	}

	for _, loadBalancer := range loadBalancersToDelete {
		log.Printf("Deleting load balancer %s (%s)", *loadBalancer.LoadBalancerName, *loadBalancer.DNSName)
		_, err := elbClient.DeleteLoadBalancer(&elb.DeleteLoadBalancerInput{
			LoadBalancerName: loadBalancer.LoadBalancerName,
		})
		if err != nil {
			log.Printf("Failed to delete load balancer %s (%s): %v", *loadBalancer.LoadBalancerName, *loadBalancer.DNSName, err)
		} else {
			log.Printf("Deleted load balancer %s (%s)", *loadBalancer.LoadBalancerName, *loadBalancer.DNSName)
			log.Printf("Sleeping a while for underlying resource is cleaned up so we could delete related security groups")
			time.Sleep(30 * time.Second)
		}

		log.Printf("Deleting security groups on load balancer %s (%s)", *loadBalancer.LoadBalancerName, *loadBalancer.DNSName)

		if loadBalancer.SourceSecurityGroup != nil {
			securityGroupName := *loadBalancer.SourceSecurityGroup.GroupName
			securityGroupId, err := GetSecurityGroupId(ec2Client, vpcId, securityGroupName)
			if err != nil {
				log.Printf("[WARN] Failed to get security group id for %s on load balancer %s: %s", securityGroupName, *loadBalancer.LoadBalancerName, err.Error())
			} else {
				common.RetryUntilTrue(func() (bool, error) {
					networkInterfaces, err := ListNetworkInterfaces(ec2Client, vpcId, securityGroupId)
					if err != nil {
						log.Printf("Failed to list network interfaces for security group %s", securityGroupId)
						return false, err
					} else {
						if len(networkInterfaces) > 0 {
							log.Printf("There is still %d network interface(s) for security group %s, waiting and checking again", len(networkInterfaces), securityGroupId)
						}
						return len(networkInterfaces) == 0, nil
					}
				},
					10*time.Minute,
					10*time.Second)

				log.Printf("Deleting source security group %s (%s) on load balancer %s", securityGroupId, securityGroupName, *loadBalancer.DNSName)
				DeleteSecurityGroupByIdIgnoreError(ec2Client, securityGroupId, 20*time.Minute)
			}
		}
		for _, securityGroupId := range loadBalancer.SecurityGroups {
			log.Printf("Deleting source security group %s on load balancer %s", *securityGroupId, *loadBalancer.DNSName)
			DeleteSecurityGroupByIdIgnoreError(ec2Client, *securityGroupId, 20*time.Minute)
		}
	}

	return nil
}

func WaitAndGetEksServiceLoadBalancerUrls(region string, kubeConfigFile string, eksClusterName string, namespace string, serviceName string) ([]string, error) {
	_, clientset, err := CreateKubernetesClient(region, kubeConfigFile, eksClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	urls, err := kubelib.GetServiceLoadBalancerUrls(clientset, namespace, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get load balancers for service %s in namespace %s in cluster %s: %v", serviceName, namespace, eksClusterName, err)
	}

	for _, url := range urls {
		session := CreateSession(region)
		elbClient := elb.New(session)
		err := common.RetryUntilTrue(func() (bool, error) {
			instanceStates, err := GetLoadBalancerInstanceStatesByDNSName(elbClient, url)
			if err != nil {
				return false, err
			}
			if len(instanceStates) == 0 {
				log.Printf("Did not find instances for load balancer %s, wait and retry", url)
				return false, nil
			}
			for _, entry := range instanceStates {
				if strings.EqualFold(*entry.State, "InService") {
					return true, nil
				}
			}
			log.Printf("No ready instance for load balancer %s, wait and retry", url)
			return false, nil
		},
			10*time.Minute,
			10*time.Second)
		if err != nil {
			return nil, fmt.Errorf("no ready instance for load balancer %s", url)
		}
	}
	return urls, nil
}
