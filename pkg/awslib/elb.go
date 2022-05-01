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
	"github.com/aws/aws-sdk-go/service/elb"
	"strings"
)

func ListLoadBalancers(elbClient *elb.ELB) ([]*elb.LoadBalancerDescription, error) {
	var result = make([]*elb.LoadBalancerDescription, 0, 100)
	var hasMoreResult = true
	var marker *string = nil
	for hasMoreResult {
		describeLoadBalancersOutput, err := elbClient.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
			Marker: marker,
		})
		if err != nil {
			return nil, err
		}
		for _, entry := range describeLoadBalancersOutput.LoadBalancerDescriptions {
			result = append(result, entry)
		}
		marker = describeLoadBalancersOutput.NextMarker
		hasMoreResult = marker != nil && *marker != ""
	}
	return result, nil
}

func GetLoadBalancerByDNSName(elbClient *elb.ELB, dnsName string) (*elb.LoadBalancerDescription, error) {
	var hasMoreResult = true
	var marker *string = nil
	for hasMoreResult {
		describeLoadBalancersOutput, err := elbClient.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
			Marker: marker,
		})
		if err != nil {
			return nil, err
		}
		for _, entry := range describeLoadBalancersOutput.LoadBalancerDescriptions {
			if strings.EqualFold(*entry.DNSName, dnsName) {
				return entry, nil
			}
		}
		marker = describeLoadBalancersOutput.NextMarker
		hasMoreResult = marker != nil && *marker != ""
	}
	return nil, nil
}

func GetLoadBalancerInstanceStatesByDnsName(elbClient *elb.ELB, dnsName string) ([]*elb.InstanceState, error) {
	var hasMoreResult = true
	var marker *string
	var loadBalancer *elb.LoadBalancerDescription
	for hasMoreResult {
		describeLoadBalancersOutput, err := elbClient.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
			Marker: marker,
		})
		if err != nil {
			return nil, err
		}
		for _, entry := range describeLoadBalancersOutput.LoadBalancerDescriptions {
			if strings.EqualFold(*entry.DNSName, dnsName) {
				loadBalancer = entry
				break
			}
		}
		if loadBalancer != nil {
			break
		}
		marker = describeLoadBalancersOutput.NextMarker
		hasMoreResult = marker != nil && *marker != ""
	}
	if loadBalancer == nil {
		return nil, fmt.Errorf("did not find load balancer %s", dnsName)
	}
	describeInstanceHealthOutput, err := elbClient.DescribeInstanceHealth(&elb.DescribeInstanceHealthInput{
		LoadBalancerName: loadBalancer.LoadBalancerName,
		Instances:        loadBalancer.Instances,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get instance health for load balancer %s (%s)", dnsName, err.Error())
	}
	return describeInstanceHealthOutput.InstanceStates, nil
}
