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

package sparkongke

import (
	"context"
	"fmt"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/common"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/gcplib"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"github.com/datapunchorg/punch/pkg/resource"
	"github.com/datapunchorg/punch/pkg/topologyimpl/gke"
	"github.com/datapunchorg/punch/pkg/topologyimpl/sparkoneks"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"strings"
	"time"
)

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*Topology)

	deployment := framework.NewDeployment()

	deployment.AddStep("createGkeCluster", "Create GKE cluster", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		err := resource.CreateGkeCluster(currentTopology.Spec.GkeSpec.ProjectId, currentTopology.Spec.GkeSpec.Location, currentTopology.Spec.GkeSpec.GkeCluster)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		return framework.DeployableOutput{}, nil
	})

	deployment.AddStep("deployNginxIngressController", "Deploy Nginx ingress controller", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		return gke.DeployNginxIngressController(*currentTopology.GetMetadata().GetCommandEnvironment(), currentTopology.Spec.GkeSpec)
	})

	deployment.AddStep("deployArgocd", "Deploy Argocd", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		return DeployArgocd(*currentTopology.GetMetadata().GetCommandEnvironment(), currentTopology.Spec)
	})

	deployment.AddStep("deploySparkOperator", "Deploy Spark Operator", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		err := DeploySparkOperator(*currentTopology.GetMetadata().GetCommandEnvironment(), currentTopology.Spec.Spark, currentTopology.Spec.GkeSpec.ProjectId, currentTopology.Spec.GkeSpec.Location, currentTopology.Spec.GkeSpec.GkeCluster.ClusterName)
		return nil, err
	})

	err := deployment.Run()
	return deployment.GetOutput(), err
}

func DeployArgocd(commandEnvironment framework.CommandEnvironment, topology TopologySpec) (map[string]interface{}, error) {
	kubeConfig, err := gcplib.CreateKubeConfig(commandEnvironment.Get(framework.CmdEnvKubeConfig), topology.GkeSpec.ProjectId, topology.GkeSpec.Location, topology.GkeSpec.GkeCluster.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get kube config: %s", err.Error())
	}
	defer kubeConfig.Cleanup()

	namespace := "argocd"

	{
		cmd := fmt.Sprintf("create namespace %s", namespace)
		arguments := strings.Split(cmd, " ")
		err = kubelib.RunKubectlWithKubeConfig(commandEnvironment.Get(framework.CmdEnvKubectlExecutable), kubeConfig, arguments)
		if err != nil {
			log.Printf("Failed to run kubectl %s: %s", cmd, err.Error())
		}
	}
	{
		cmd := fmt.Sprintf("apply -n %s -f %s", namespace, topology.ArgocdInstallYamlFile)
		arguments := strings.Split(cmd, " ")
		err = kubelib.RunKubectlWithKubeConfig(commandEnvironment.Get(framework.CmdEnvKubectlExecutable), kubeConfig, arguments)
		if err != nil {
			return nil, err
		}
	}

	_, clientset, err := gcplib.CreateKubernetesClient(commandEnvironment.Get(framework.CmdEnvKubeConfig), topology.GkeSpec.ProjectId, topology.GkeSpec.Location, topology.GkeSpec.GkeCluster.ClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	podNamePrefix := "argocd-server"
	err = kubelib.WaitPodsInPhases(clientset, namespace, podNamePrefix, []v1.PodPhase{v1.PodRunning})
	if err != nil {
		return nil, fmt.Errorf("pod %s*** in namespace %s is not in phase %s", podNamePrefix, namespace, v1.PodRunning)
	}

	output := make(map[string]interface{})

	return output, nil
}

func DeploySparkOperator(commandEnvironment framework.CommandEnvironment, sparkComponentSpec sparkoneks.SparkComponentSpec, projectId, zone, clusterName string) error {
	operatorNamespace := sparkComponentSpec.Operator.Namespace

	/*eventLogDir := sparkComponentSpec.Gateway.SparkEventLogDir
	if strings.HasPrefix(strings.ToLower(eventLogDir), "s3") {
		dummyFileUrl := eventLogDir
		if !strings.HasSuffix(dummyFileUrl, "/") {
			dummyFileUrl += "/"
		}
		dummyFileUrl += "dummy.txt"
		log.Printf("Uploading dummy file %s to Spark event log directory %s to make sure the directry exits (Spark application and history server will fail if the directory does not exist)", dummyFileUrl, eventLogDir)
		err := awslib.UploadDataToS3Url(region, dummyFileUrl, strings.NewReader(""))
		if err != nil {
			return fmt.Errorf("failed to create dummy file %s in Spark event log directory %s to make sure the directry exits (Spark application and history server will fail if the directory does not exist): %s", dummyFileUrl, eventLogDir, err.Error())
		}
	}*/

	_, clientset, err := gcplib.CreateKubernetesClient(commandEnvironment.Get(framework.CmdEnvKubeConfig), projectId, zone, clusterName)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	sparkApplicationNamespace := sparkComponentSpec.Operator.SparkApplicationNamespace

	namespace, err := clientset.CoreV1().Namespaces().Create(
		context.TODO(),
		&v1.Namespace{
			ObjectMeta: v12.ObjectMeta{
				Name: sparkApplicationNamespace,
			},
		},
		v12.CreateOptions{},
	)
	if err != nil {
		if !awslib.AlreadyExistsMessage(err.Error()) { // TODO check error more precisely
			return fmt.Errorf("failed to create Spark application namespace %s: %v", sparkApplicationNamespace, err)
		} else {
			log.Printf("Namespace %s already exists, do not create it again", sparkApplicationNamespace)
		}
	} else {
		log.Printf("Created Spark application namespace %s", namespace.Name)
	}

	err = InstallSparkOperatorHelm(commandEnvironment, sparkComponentSpec, projectId, zone, clusterName)
	if err != nil {
		return err
	}

	// CreateSparkServiceAccount(clientset, operatorNamespace, sparkApplicationNamespace, "spark")

	helmInstallName := sparkComponentSpec.Operator.HelmInstallName
	err = sparkoneks.CreateApiGatewayService(clientset, operatorNamespace, helmInstallName, helmInstallName)
	if err != nil {
		return err
	}

	sparkOperatorPodNamePrefix := helmInstallName
	err = kubelib.WaitPodsInPhases(clientset, operatorNamespace, sparkOperatorPodNamePrefix, []v1.PodPhase{v1.PodRunning})
	if err != nil {
		return fmt.Errorf("pod %s*** in namespace %s is not in phase %s", sparkOperatorPodNamePrefix, operatorNamespace, v1.PodRunning)
	}

	// Retry and handle error like following
	// Failed to create ingress spark-operator-01 in namespace spark-operator-01 for service spark-operator-01: Internal error occurred: failed calling webhook "validate.nginx.ingress.kubernetes.io": Post "https://ingress-nginx-controller-admission.ingress-nginx.svc:443/networking/v1/ingresses?timeout=10s": context deadline exceeded
	return common.RetryUntilTrue(func() (bool, error) {
		err := sparkoneks.CreateApiGatewayIngress(clientset, operatorNamespace, helmInstallName, helmInstallName, sparkComponentSpec.Gateway.IngressHot)
		ignoreErrorMsg := "failed calling webhook \"validate.nginx.ingress.kubernetes.io\""
		if err != nil {
			if strings.Contains(err.Error(), ignoreErrorMsg) {
				log.Printf("Ignore error in creating ingress and will retry: %s", err.Error())
				return false, nil
			} else {
				return false, err
			}
		} else {
			return true, nil
		}
	},
		10*time.Minute,
		10*time.Second)
}

func InstallSparkOperatorHelm(commandEnvironment framework.CommandEnvironment, sparkComponentSpec sparkoneks.SparkComponentSpec, projectId, zone, clusterName string) error {
	// helm install my-release spark-operator/spark-operator --namespace spark-operator --create-namespace --set sparkJobNamespace=default

	region := "gcp:us-central1" // TODO remove this
	s3Bucket := "todo"          // TODO remove this

	kubeConfig, err := gcplib.CreateKubeConfig(commandEnvironment.Get(framework.CmdEnvKubeConfig), projectId, zone, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := sparkComponentSpec.Operator.HelmInstallName
	operatorNamespace := sparkComponentSpec.Operator.Namespace
	sparkApplicationNamespace := sparkComponentSpec.Operator.SparkApplicationNamespace

	arguments := []string{
		"--set", fmt.Sprintf("sparkJobNamespace=%s", sparkApplicationNamespace),
		"--set", fmt.Sprintf("image.repository=%s", sparkComponentSpec.Operator.ImageRepository),
		"--set", fmt.Sprintf("image.tag=%s", sparkComponentSpec.Operator.ImageTag),
		"--set", "serviceAccounts.spark.create=true",
		"--set", "serviceAccounts.spark.name=spark",
		"--set", "spark.gateway.user=" + sparkComponentSpec.Gateway.User,
		"--set", "spark.gateway.password=" + sparkComponentSpec.Gateway.Password,
		"--set", "spark.gateway.s3Region=" + region,
		"--set", "spark.gateway.s3Bucket=" + s3Bucket,
		// "--set", "webhook.enable=true",
	}

	if !commandEnvironment.GetBoolOrElse(framework.CmdEnvWithMinikube, false) {
		arguments = append(arguments, "--set")
		arguments = append(arguments, "spark.gateway.sparkEventLogEnabled=true")
		arguments = append(arguments, "--set")
		arguments = append(arguments, "spark.gateway.sparkEventLogDir="+sparkComponentSpec.Gateway.SparkEventLogDir)
	}

	if sparkComponentSpec.Gateway.HiveMetastoreUris != "" {
		arguments = append(arguments, "--set")
		arguments = append(arguments, "spark.gateway.hiveMetastoreUris="+sparkComponentSpec.Gateway.HiveMetastoreUris)
	}
	if sparkComponentSpec.Gateway.SparkSqlWarehouseDir != "" {
		arguments = append(arguments, "--set")
		arguments = append(arguments, "spark.gateway.sparkSqlWarehouseDir="+sparkComponentSpec.Gateway.SparkSqlWarehouseDir)
	}

	err = kubelib.InstallHelm(commandEnvironment.Get(framework.CmdEnvHelmExecutable), commandEnvironment.Get(sparkoneks.CmdEnvSparkOperatorHelmChart), kubeConfig, arguments, installName, operatorNamespace)
	return err
}
