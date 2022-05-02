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
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/common"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"github.com/datapunchorg/punch/pkg/resource"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var NodePortLocalHttp int32 = 32080
var NodePortLocalHttps int32 = 32443

func CreateInstanceIamRole(topology EksTopologySpec) string {
	region := topology.Region
	roleName, err := resource.CreateIAMRoleWithMorePolicies(region, topology.Eks.InstanceRole, []resource.IAMPolicy{topology.S3Policy})
	if err != nil {
		// TODO remove Fatalf
		log.Fatalf("Failed to create instance IAM role: %s", err.Error())
	}
	return roleName
}

// CreateClusterAutoscalerIamRole returns IAM Role name like: AmazonEKSClusterAutoscalerRole-xxx
func CreateClusterAutoscalerIamRole(topology EksTopologySpec, oidcId string) (string, error) {
	region := topology.Region
	session := awslib.CreateSession(region)
	accountId, err := awslib.GetCurrentAccount(session)
	if err != nil {
		return "", fmt.Errorf("failed to get current account: %s", err.Error())
	}
	role := topology.AutoScaling.ClusterAutoscalerIAMRole
	role.AssumeRolePolicyDocument = fmt.Sprintf(`{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "Federated": "arn:aws:iam::%s:oidc-provider/oidc.eks.us-west-1.amazonaws.com/id/%s"
            },
            "Action": "sts:AssumeRoleWithWebIdentity",
            "Condition": {
                "StringEquals": {
                    "oidc.eks.us-west-1.amazonaws.com/id/%s:aud": "sts.amazonaws.com",
                    "oidc.eks.us-west-1.amazonaws.com/id/%s:sub": "system:serviceaccount:kube-system:cluster-autoscaler"
                }
            }
        }
    ]
}
`, accountId, oidcId, oidcId, oidcId)
	role.Name += "-" + oidcId
	err = resource.CreateIAMRole(region, role)
	if err != nil {
		return "", fmt.Errorf("failed to create cluster autoscaler IAM role: %s", err.Error())
	}
	return role.Name, nil
}

func CreateClusterAutoscalerIamServiceAccount(commandEnvironment framework.CommandEnvironment, topology EksTopologySpec, iamRoleName string) error {
	region := topology.Region
	session := awslib.CreateSession(region)
	accountId, err := awslib.GetCurrentAccount(session)
	if err != nil {
		return fmt.Errorf("failed to get current account: %s", err.Error())
	}
	roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, iamRoleName)
	systemNamespace := "kube-system"
	serviceAccountName := "cluster-autoscaler"
	awslib.RunEksCtlCmd("eksctl",
		[]string{"create", "iamserviceaccount",
			"--name", serviceAccountName,
			"--region", topology.Region,
			"--cluster", topology.Eks.ClusterName,
			"--namespace", systemNamespace,
			"--attach-role-arn", roleArn,
			"--approve"})
	err = WaitEksServiceAccount(commandEnvironment, topology, systemNamespace, serviceAccountName)
	return err
}

func WaitEksServiceAccount(commandEnvironment framework.CommandEnvironment, topology EksTopologySpec, namespace string, serviceAccount string) error {
	region := topology.Region
	clusterName := topology.Eks.ClusterName

	_, clientset, err := awslib.CreateKubernetesClient(region, commandEnvironment.Get(CmdEnvKubeConfig), clusterName)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}
	err = common.RetryUntilTrue(func() (bool, error) {
		_, err = clientset.CoreV1().ServiceAccounts(namespace).Get(
			context.TODO(),
			serviceAccount,
			v12.GetOptions{},
		)
		if err != nil {
			fmt.Sprintf("Failed to get service account %s in namespace %s in Eks cluster %s", serviceAccount, namespace, clusterName)
			return false, nil
		}
		return true, nil
	},
		5*time.Minute,
		10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to wait service account %s in namespace %s in Eks cluster %s", serviceAccount, namespace, clusterName)
	}
	return nil
}

// TODO remove log.Fatalf
func DeployNginxIngressController(commandEnvironment framework.CommandEnvironment, topology EksTopologySpec) map[string]interface{} {
	nginxNamespace := topology.NginxIngress.Namespace
	helmInstallName := topology.NginxIngress.HelmInstallName
	serviceName := "ingress-nginx-controller"
	region := topology.Region
	eksClusterName := topology.Eks.ClusterName
	kubeConfig, err := awslib.CreateKubeConfig(region, commandEnvironment.Get(CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		log.Fatalf("Failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	arguments := []string{
		"--set", fmt.Sprintf("controller.service.enableHttp=%t", topology.NginxIngress.EnableHttp),
		"--set", fmt.Sprintf("controller.service.enableHttps=%t", topology.NginxIngress.EnableHttps),
	}

	if commandEnvironment.GetBoolOrElse(CmdEnvWithMinikube, false) {
		arguments = append(arguments, "--set", "controller.service.type=NodePort")
		arguments = append(arguments, "--set", fmt.Sprintf("controller.service.nodePorts.http=%d", NodePortLocalHttp))
		arguments = append(arguments, "--set", fmt.Sprintf("controller.service.nodePorts.https=%d", NodePortLocalHttps))
	}

	kubelib.InstallHelm(commandEnvironment.Get(CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvNginxHelmChart), kubeConfig, arguments, helmInstallName, nginxNamespace)

	_, clientset, err := awslib.CreateKubernetesClient(region, commandEnvironment.Get(CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %s", err.Error())
	}
	err = kubelib.WaitPodsInPhase(clientset, nginxNamespace, serviceName, v1.PodRunning)
	if err != nil {
		log.Fatalf("Pod %s*** in namespace %s is not in phase %s", serviceName, nginxNamespace, v1.PodRunning)
	}

	hostPorts, err := awslib.GetLoadBalancerHostPorts(region, commandEnvironment.Get(CmdEnvKubeConfig), eksClusterName, nginxNamespace, serviceName)
	if err != nil {
		log.Fatalf("Failed to get load balancer urls for nginx controller service %s in namespace %s", serviceName, nginxNamespace)
	}

	dnsNamesMap := make(map[string]bool, len(hostPorts))
	for _, entry := range hostPorts {
		dnsNamesMap[entry.Host] = true
	}
	dnsNames := make([]string, 0, len(dnsNamesMap))
	for k := range dnsNamesMap {
		dnsNames = append(dnsNames, k)
	}

	if !commandEnvironment.GetBoolOrElse(CmdEnvWithMinikube, false) {
		err = awslib.WaitLoadBalancersReadyByDnsNames(region, dnsNames)
		if err != nil {
			log.Fatalf("Failed to wait and get load balancer urls: %s", err.Error())
		}
	}

	urls := make([]string, 0, len(hostPorts))
	for _, entry := range hostPorts {
		if entry.Port == 443 {
			urls = append(urls, fmt.Sprintf("https://%s", entry.Host))
		} else if entry.Port == NodePortLocalHttps {
			urls = append(urls, fmt.Sprintf("https://%s:%d", entry.Host, entry.Port))
		} else if entry.Port == 80 {
			urls = append(urls, fmt.Sprintf("http://%s", entry.Host))
		} else {
			urls = append(urls, fmt.Sprintf("http://%s:%d", entry.Host, entry.Port))
		}
	}
	output := make(map[string]interface{})
	output["loadBalancerUrls"] = urls

	preferredUrl := ""
	if len(urls) > 0 {
		preferredUrl = urls[0]
		for _, entry := range urls {
			// Use https url if possible
			if strings.HasPrefix(strings.ToLower(entry), "https") {
				preferredUrl = entry
			}
		}
	}

	output["loadBalancerPreferredUrl"] = preferredUrl

	return output
}

func DeleteEksCluster(region string, clusterName string) {
	err := awslib.DeleteEKSCluster(region, clusterName)
	if err != nil {
		fmt.Sprintf("Failed to delete Eks cluster: %s", err.Error())
	}
	err = awslib.CheckEksCtlCmd("eksctl")
	if err == nil {
		awslib.RunEksCtlCmd("eksctl",
			[]string{"delete", "iamserviceaccount",
				"--region", region,
				"--cluster", clusterName,
				"--namespace", "kube-system",
				"--name", "cluster-autoscaler",
				"--wait",
			})
		awslib.RunEksCtlCmd("eksctl",
			[]string{"delete", "cluster",
				"--region", region,
				"--name", clusterName,
			})
	}
}
