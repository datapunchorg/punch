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

package eks

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/common"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"github.com/datapunchorg/punch/pkg/resource"
	v1 "k8s.io/api/core/v1"
	"log"
	"strings"
	"time"
)

func CreateInstanceIAMRole(topology EksTopology) string {
	region := topology.Spec.Region
	roleName, err := resource.CreateIAMRoleWithMorePolicies(region, topology.Spec.EKS.InstanceRole, []resource.IAMPolicy{topology.Spec.S3Policy})
	if err != nil {
		log.Fatalf("Failed to create instance IAM role: %s", err.Error())
	}
	return roleName
}

func DeployNginxIngressController(commandEnvironment framework.CommandEnvironment, topology EksTopology) map[string]interface{} {
	nginxNamespace := topology.Spec.NginxIngress.Namespace
	helmInstallName := topology.Spec.NginxIngress.HelmInstallName
	serviceName := "ingress-nginx-controller"
	region := topology.Spec.Region
	eksClusterName := topology.Spec.EKS.ClusterName
	kubeConfig, err := awslib.CreateKubeConfig(region, commandEnvironment.Get(CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		log.Fatalf("Failed to get kube config: %s", err)
	}

	arguments := []string{
		"--set", fmt.Sprintf("service.enableHttp=%t", topology.Spec.NginxIngress.EnableHttp),
		"--set", fmt.Sprintf("service.enableHttps=%t", topology.Spec.NginxIngress.EnableHttps),
	}

	kubelib.InstallHelm(commandEnvironment.Get(CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvNginxHelmChart), kubeConfig, arguments, helmInstallName, nginxNamespace)

	_, clientset, err := awslib.CreateKubernetesClient(region, commandEnvironment.Get(CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}
	err = kubelib.WaitPodsInPhase(clientset, nginxNamespace, serviceName, v1.PodRunning)
	if err != nil {
		log.Fatalf("Pod %s* in namespace %s is not in phase %s", serviceName, nginxNamespace, v1.PodRunning)
	}

	urls, err := kubelib.GetServiceLoadBalancerUrls(clientset, nginxNamespace, serviceName)
	if err != nil {
		log.Fatalf("Failed to get load balancers for service %s in namespace %s in cluster %s: %v", serviceName, nginxNamespace, eksClusterName, err)
	}

	for _, url := range urls {
		session := awslib.CreateSession(region)
		elbClient := elb.New(session)
		common.RetryUntilTrue(func() (bool, error) {
			instanceStates, err := awslib.GetLoadBalancerInstanceStatesByDNSName(elbClient, url)
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
	}

	output := make(map[string]interface{})
	output["loadBalancerUrls"] = urls
	return output
}

