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
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/datapunchorg/punch/pkg/common"
	"github.com/datapunchorg/punch/pkg/kubelib"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"strings"
	"time"
)

type LoadBalancerInfo struct {
	LoadBalancerName        string
	DNSName                 string
	SourceSecurityGroupName string
	SecurityGroupIds        []string
}

func DeleteAllLoadBalancersOnEks(region string, vpcId string, eksClusterName string) error {
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

	namespaceList, err := clientset.CoreV1().Namespaces().List(context.TODO(), v12.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list namespaces for cluster %s: %s", eksClusterName, err.Error())
	}

	nonSystemNamespaces := make([]string, 0, len(namespaceList.Items))
	for _, entry := range namespaceList.Items {
		str := strings.ToLower(entry.Name)
		if strings.HasPrefix(str, "kube-") {
			continue
		}
		nonSystemNamespaces = append(nonSystemNamespaces, entry.Name)
	}

	for _, nonSystemNamespace := range nonSystemNamespaces {
		log.Printf("Checking and deleting load balancers in namespace %s, EKS: %s, region: %s, VPC: %s", nonSystemNamespace, eksClusterName, region, vpcId)
		err = deleteLoadBalancersOnEks(session, ec2Client, clientset, vpcId, nonSystemNamespace)
		if err != nil {
			return err
		}
	}

	return nil
}

func DeleteLoadBalancersOnEks(region string, vpcId string, eksClusterName string, namespace string) error {
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

	return deleteLoadBalancersOnEks(session, ec2Client, clientset, vpcId, namespace)
}

func deleteLoadBalancersOnEks(session *session.Session, ec2Client *ec2.EC2, clientset *kubernetes.Clientset, vpcId string, namespace string) error {
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
		log.Printf("Did not find load balancer in namepsace %s, do not delete any load balancer there", namespace)
		return nil
	}

	loadBalancersToDelete := make([]LoadBalancerInfo, 0, 100)

	elbClient := elb.New(session)
	loadBalancers, err := ListLoadBalancers(elbClient)
	if err != nil {
		return fmt.Errorf("failed to list load balancers in namespace %s: %s", namespace, err.Error())
	}
	for _, hostName := range ingressHostNames {
		for _, loadBalancer := range loadBalancers {
			if strings.EqualFold(*loadBalancer.DNSName, hostName) {
				loadBalancerInfo := LoadBalancerInfo{
					LoadBalancerName: *loadBalancer.LoadBalancerName,
					DNSName:          *loadBalancer.DNSName,
				}
				if loadBalancer.SourceSecurityGroup != nil {
					loadBalancerInfo.SourceSecurityGroupName = *loadBalancer.SourceSecurityGroup.GroupName
				}
				if loadBalancer.SecurityGroups != nil {
					loadBalancerInfo.SecurityGroupIds = make([]string, 0, len(loadBalancer.SecurityGroups))
					for _, entry := range loadBalancer.SecurityGroups {
						loadBalancerInfo.SecurityGroupIds = append(loadBalancerInfo.SecurityGroupIds, *entry)
					}
				}
				loadBalancersToDelete = append(loadBalancersToDelete, loadBalancerInfo)
			}
		}
	}

	elbClientV2 := elbv2.New(session)
	loadBalancersV2, err := ListLoadBalancersV2(elbClientV2)
	if err != nil {
		return fmt.Errorf("failed to list load balancers in namespace %s: %s", namespace, err.Error())
	}
	for _, hostName := range ingressHostNames {
		for _, loadBalancer := range loadBalancersV2 {
			if strings.EqualFold(*loadBalancer.DNSName, hostName) {
				loadBalancerInfo := LoadBalancerInfo{
					LoadBalancerName: *loadBalancer.LoadBalancerName,
					DNSName:          *loadBalancer.DNSName,
				}
				if loadBalancer.SecurityGroups != nil {
					loadBalancerInfo.SecurityGroupIds = make([]string, 0, len(loadBalancer.SecurityGroups))
					for _, entry := range loadBalancer.SecurityGroups {
						loadBalancerInfo.SecurityGroupIds = append(loadBalancerInfo.SecurityGroupIds, *entry)
					}
				}
				loadBalancersToDelete = append(loadBalancersToDelete, loadBalancerInfo)
			}
		}
	}

	for _, loadBalancer := range loadBalancersToDelete {
		log.Printf("Deleting load balancer %s (%s)", loadBalancer.LoadBalancerName, loadBalancer.DNSName)
		_, err := elbClient.DeleteLoadBalancer(&elb.DeleteLoadBalancerInput{
			LoadBalancerName: aws.String(loadBalancer.LoadBalancerName),
		})
		if err != nil {
			log.Printf("Failed to delete load balancer %s (%s): %s", loadBalancer.LoadBalancerName, loadBalancer.DNSName, err.Error())
		} else {
			log.Printf("Deleted load balancer %s (%s)", loadBalancer.LoadBalancerName, loadBalancer.DNSName)
			log.Printf("Sleeping a while for underlying resource is cleaned up so we could delete related security groups")
			time.Sleep(30 * time.Second)
		}

		log.Printf("Deleting security groups on load balancer %s (%s)", loadBalancer.LoadBalancerName, loadBalancer.DNSName)

		if loadBalancer.SourceSecurityGroupName != "" {
			securityGroupName := loadBalancer.SourceSecurityGroupName
			securityGroupId, err := GetSecurityGroupId(ec2Client, vpcId, securityGroupName)
			if err != nil {
				log.Printf("[WARN] Failed to get security group id for %s on load balancer %s: %s", securityGroupName, loadBalancer.LoadBalancerName, err.Error())
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

				log.Printf("Deleting source security group %s (%s) on load balancer %s", securityGroupId, securityGroupName, loadBalancer.DNSName)
				err := ListAndDeleteSecurityGroupRulesBySourceOrDestinationSecurityGroupId(ec2Client, vpcId, securityGroupId)
				if err != nil {
					log.Printf("[WARN] failed to delete security group rules by source or destination security group id %s: %s", securityGroupId, err.Error())
				}
				DeleteSecurityGroupByIdIgnoreError(ec2Client, securityGroupId, 20*time.Minute)
			}
		}
		for _, securityGroupId := range loadBalancer.SecurityGroupIds {
			log.Printf("Deleting security group %s on load balancer %s", securityGroupId, loadBalancer.DNSName)
			err := ListAndDeleteSecurityGroupRulesBySourceOrDestinationSecurityGroupId(ec2Client, vpcId, securityGroupId)
			if err != nil {
				log.Printf("[WARN] failed to delete security group rules by source or destination security group id %s: %s", securityGroupId, err.Error())
			}
			DeleteSecurityGroupByIdIgnoreError(ec2Client, securityGroupId, 20*time.Minute)
		}
	}

	return nil
}

func GetServiceLoadBalancerHostPorts(region string, kubeConfigFile string, eksClusterName string, namespace string, serviceName string) ([]common.HostPort, error) {
	_, clientset, err := CreateKubernetesClient(region, kubeConfigFile, eksClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}
	hostPorts, err := kubelib.GetServiceLoadBalancerHostPorts(clientset, namespace, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get load balancers for service %s in namespace %s in cluster %s: %v", serviceName, namespace, eksClusterName, err)
	}
	return hostPorts, nil
}

func WaitServiceLoadBalancerHostPorts(region string, kubeConfig string, eksClusterName string, namespace string, serviceName string) ([]common.HostPort, error) {
	hostPorts, err := GetServiceLoadBalancerHostPorts(region, kubeConfig, eksClusterName, namespace, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get load balancer urls for service %s in namespace %s", serviceName, namespace)
	}

	dnsNameSet := make(map[string]bool, len(hostPorts))
	for _, entry := range hostPorts {
		dnsNameSet[entry.Host] = true
	}
	dnsNames := make([]string, 0, len(dnsNameSet))
	for k := range dnsNameSet {
		dnsNames = append(dnsNames, k)
	}
	err = WaitLoadBalancersReadyByDnsNames(region, dnsNames)
	if err != nil {
		return nil, fmt.Errorf("failed to wait and get load balancer urls: %s", err.Error())
	}
	return hostPorts, nil
}

func WaitLoadBalancersReadyByDnsNames(region string, dnsNames []string) error {
	for _, dnsName := range dnsNames {
		session := CreateSession(region)
		elbClient := elbv2.New(session)
		err := common.RetryUntilTrue(func() (bool, error) {
			state, err := GetLoadBalancerStatesByDnsNameV2(elbClient, dnsName)
			if err != nil {
				return false, err
			}
			if strings.EqualFold(*state.Code, "Active") {
				return true, nil
			}
			log.Printf("No active for load balancer %s, wait and retry", dnsName)
			return false, nil
		},
			10*time.Minute,
			10*time.Second)
		if err != nil {
			return fmt.Errorf("no ready instance for load balancer %s", dnsName)
		}
	}
	return nil
}
