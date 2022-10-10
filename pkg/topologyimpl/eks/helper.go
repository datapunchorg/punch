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

package eks

import (
	"context"
	"encoding/json"
	"fmt"
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
var DefaultNginxServiceName = "ingress-nginx-controller"

func ValidateEksTopologySpec(spec EksTopologySpec, metadata framework.TopologyMetadata, phase string) error {
	err := framework.CheckCmdEnvFolderExists(metadata, CmdEnvNginxHelmChart)
	if err != nil {
		return err
	}
	if spec.AutoScaling.EnableClusterAutoscaler {
		err := framework.CheckCmdEnvFolderExists(metadata, CmdEnvClusterAutoscalerHelmChart)
		if err != nil {
			return err
		}
		err = awslib.CheckEksCtlCmd("eksctl")
		if err != nil {
			return fmt.Errorf("cluster autoscaler enabled, but cannot find eksctl command: %s", err.Error())
		}
	}
	return nil
}

func CreateInstanceIamRole(topology EksTopologySpec) (string, error) {
	region := topology.Region
	roleName, err := resource.CreateIamRoleWithMorePolicies(region, topology.EksCluster.InstanceRole, []resource.IamPolicy{topology.S3Policy, topology.KafkaPolicy})
	if err != nil {
		return "", fmt.Errorf("failed to create instance IAM role: %s", err.Error())
	}
	return roleName, nil
}

// CreateClusterAutoscalerIamRole returns IAM Role name like: AmazonEKSClusterAutoscalerRole-xxx
func CreateClusterAutoscalerIamRole(topology EksTopologySpec, oidcId string) (string, error) {
	region := topology.Region
	session := awslib.CreateSession(region)
	accountId, err := awslib.GetCurrentAccount(session)
	if err != nil {
		return "", fmt.Errorf("failed to get current account: %s", err.Error())
	}
	role := topology.AutoScaling.ClusterAutoscalerIamRole
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
	err = resource.CreateIamRole(region, role)
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
			"--cluster", topology.EksCluster.ClusterName,
			"--namespace", systemNamespace,
			"--attach-role-arn", roleArn,
			"--approve"})
	err = WaitEksServiceAccount(commandEnvironment, topology, systemNamespace, serviceAccountName)
	return err
}

func WaitEksServiceAccount(commandEnvironment framework.CommandEnvironment, topology EksTopologySpec, namespace string, serviceAccount string) error {
	region := topology.Region
	clusterName := topology.EksCluster.ClusterName

	_, clientset, err := awslib.CreateKubernetesClient(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), clusterName)
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
			fmt.Sprintf("Failed to get service account %s in namespace %s in EKS cluster %s", serviceAccount, namespace, clusterName)
			return false, nil
		}
		return true, nil
	},
		5*time.Minute,
		10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to wait service account %s in namespace %s in EKS cluster %s", serviceAccount, namespace, clusterName)
	}
	return nil
}

func DeployNginxIngressController(commandEnvironment framework.CommandEnvironment, topology EksTopologySpec) (map[string]interface{}, error) {
	nginxNamespace := topology.NginxIngress.Namespace
	helmInstallName := topology.NginxIngress.HelmInstallName
	nginxServiceName := DefaultNginxServiceName
	region := topology.Region
	eksClusterName := topology.EksCluster.ClusterName
	kubeConfig, err := awslib.CreateKubeConfig(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get kube config: %s", err.Error())
	}

	defer kubeConfig.Cleanup()

	arguments := []string{
		"--set", fmt.Sprintf("controller.service.enableHttp=%t", topology.NginxIngress.EnableHttp),
		"--set", fmt.Sprintf("controller.service.enableHttps=%t", topology.NginxIngress.EnableHttps),
	}

	if topology.NginxIngress.HttpsBackendPort != "" {
		arguments = append(arguments, "--set", fmt.Sprintf("controller.service.targetPorts.https=%s", topology.NginxIngress.HttpsBackendPort))
	}

	if len(topology.NginxIngress.Annotations) > 0 {
		annotationsJsonBytes, err := json.Marshal(topology.NginxIngress.Annotations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal topology.NginxIngress.Annotations: %s", err.Error())
		}
		annotationsJson := string(annotationsJsonBytes)
		arguments = append(arguments, "--set-json")
		arguments = append(arguments, fmt.Sprintf("controller.service.annotations=%s", annotationsJson))
	}

	if commandEnvironment.GetBoolOrElse(framework.CmdEnvWithMinikube, false) {
		arguments = append(arguments, "--set", "controller.service.type=NodePort")
		arguments = append(arguments, "--set", fmt.Sprintf("controller.service.nodePorts.http=%d", NodePortLocalHttp))
		arguments = append(arguments, "--set", fmt.Sprintf("controller.service.nodePorts.https=%d", NodePortLocalHttps))
	}

	kubelib.InstallHelm(commandEnvironment.Get(framework.CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvNginxHelmChart), kubeConfig, arguments, helmInstallName, nginxNamespace)

	_, clientset, err := awslib.CreateKubernetesClient(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}
	err = kubelib.WaitPodsInPhases(clientset, nginxNamespace, nginxServiceName, []v1.PodPhase{v1.PodRunning})
	if err != nil {
		return nil, fmt.Errorf("pod %s*** in namespace %s is not in phase %s", nginxServiceName, nginxNamespace, v1.PodRunning)
	}

	urls, err := resource.GetEksNginxLoadBalancerUrls(commandEnvironment, region, eksClusterName, nginxNamespace, nginxServiceName, NodePortLocalHttps)
	if err != nil {
		return nil, fmt.Errorf("failed to get NGINX load balancer urls: %s", err.Error())
	}

	output := make(map[string]interface{})
	output["loadBalancerUrls"] = urls

	preferredUrl := resource.GetLoadBalancerPreferredUrl(urls)
	output["loadBalancerPreferredUrl"] = preferredUrl

	return output, nil
}

func DeleteEksCluster(region string, clusterName string) {
	err := awslib.DeleteEKSCluster(region, clusterName)
	if err != nil {
		fmt.Sprintf("Failed to delete EKS cluster: %s", err.Error())
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
