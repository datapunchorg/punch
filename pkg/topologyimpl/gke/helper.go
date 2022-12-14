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

package gke

var NodePortLocalHttp int32 = 32080
var NodePortLocalHttps int32 = 32443
var DefaultNginxServiceName = "ingress-nginx-controller"

/*
func CreateInstanceIamRole(topology TopologySpec) (string, error) {
	region := topology.Region
	roleName, err := resource.CreateIamRoleWithMorePolicies(region, topology.EksCluster.InstanceRole, []resource.IamPolicy{topology.S3Policy, topology.KafkaPolicy})
	if err != nil {
		return "", fmt.Errorf("failed to create instance IAM role: %s", err.Error())
	}
	return roleName, nil
}

// CreateClusterAutoscalerIamRole returns IAM Role name like: AmazonEKSClusterAutoscalerRole-xxx
func CreateClusterAutoscalerIamRole(topology TopologySpec, oidcId string) (string, error) {
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

func CreateClusterAutoscalerIamServiceAccount(commandEnvironment framework.CommandEnvironment, topology TopologySpec, iamRoleName string) error {
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

func WaitEksServiceAccount(commandEnvironment framework.CommandEnvironment, topology TopologySpec, namespace string, serviceAccount string) error {
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
*/
