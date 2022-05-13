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

package pinotquickstart

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/common"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	v1 "k8s.io/api/core/v1"
	"log"
	"strings"
	"time"
)

func DeployPinotService(commandEnvironment framework.CommandEnvironment, pinotComponentSpec PinotComponentSpec, region string, eksClusterName string) (string, error) {
	_, clientset, err := awslib.CreateKubernetesClient(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return "", fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	err = InstallPinotHelm(commandEnvironment, pinotComponentSpec, region, eksClusterName)
	if err != nil {
		return "", fmt.Errorf("failed to install Spark History Server helm chart: %s", err.Error())
	}

	namespace := pinotComponentSpec.Namespace
	podNamePrefix := "pinot-broker-0"
	err = kubelib.WaitPodsInPhases(clientset, namespace, podNamePrefix, []v1.PodPhase{v1.PodRunning})
	if err != nil {
		return "", fmt.Errorf("pod %s*** in namespace %s is not in phase %s", podNamePrefix, namespace, v1.PodRunning)
	}

	serviceName := "pinot-controller"
	hostPorts, err := awslib.WaitServiceLoadBalancerHostPorts(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName, namespace, serviceName)
	if err != nil {
		return "", err
	}

	for _, entry := range hostPorts {
		// TODO do not hard code port 9000
		if entry.Port == 9000 {
			url := fmt.Sprintf("http://%s:%d", entry.Host, entry.Port)
			return url, nil
		}
	}

	return "", fmt.Errorf("did not find load balancer url for %s service in namespace %s", serviceName, namespace)
}

func InstallPinotHelm(commandEnvironment framework.CommandEnvironment, pinotComponentSpec PinotComponentSpec, region string, eksClusterName string) error {
	kubeConfig, err := awslib.CreateKubeConfig(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := pinotComponentSpec.HelmInstallName
	installNamespace := pinotComponentSpec.Namespace

	arguments := []string {
		"--set",
		"controller.service.type=LoadBalancer",
	}

	err = kubelib.InstallHelm(commandEnvironment.Get(framework.CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvPinotHelmChart), kubeConfig, arguments, installName, installNamespace)
	return err
}

func DeployKafkaService(commandEnvironment framework.CommandEnvironment, kafkaComponentSpec KafkaComponentSpec, region string, eksClusterName string) error {
	_, clientset, err := awslib.CreateKubernetesClient(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	err = InstallKafkaHelm(commandEnvironment, kafkaComponentSpec, region, eksClusterName)
	if err != nil {
		return fmt.Errorf("failed to install Spark History Server helm chart: %s", err.Error())
	}

	namespace := kafkaComponentSpec.Namespace
	podNamePrefix := "kafka-2"
	err = kubelib.WaitPodsInPhases(clientset, namespace, podNamePrefix, []v1.PodPhase{v1.PodRunning})
	if err != nil {
		return fmt.Errorf("pod %s*** in namespace %s is not in phase %s", podNamePrefix, namespace, v1.PodRunning)
	}

	return nil
}

func InstallKafkaHelm(commandEnvironment framework.CommandEnvironment, kafkaComponentSpec KafkaComponentSpec, region string, eksClusterName string) error {
	kubeConfig, err := awslib.CreateKubeConfig(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := kafkaComponentSpec.HelmInstallName
	installNamespace := kafkaComponentSpec.Namespace

	arguments := make([]string, 0, 100)

	err = kubelib.InstallHelm(commandEnvironment.Get(framework.CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvKafkaHelmChart), kubeConfig, arguments, installName, installNamespace)
	return err
}

func CreateKafkaTopics(commandEnvironment framework.CommandEnvironment, region string, eksClusterName string) error {
	return common.RetryUntilTrue(func() (bool, error) {
		err := CreateKafkaTopicsNoRetry(commandEnvironment, region, eksClusterName)
		if err != nil {
			log.Printf("Failed to create kafka topics and will retry: %s", err.Error())
			return false, nil
		}
		return true, nil
	},
		5*time.Minute,
		10*time.Second)
}

func CreateKafkaTopicsNoRetry(commandEnvironment framework.CommandEnvironment, region string, eksClusterName string) error {
	kubeConfig, err := awslib.CreateKubeConfig(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %s", err.Error())
	}
	defer kubeConfig.Cleanup()

	cmd := "-n pinot-quickstart exec kafka-0 -- kafka-topics --zookeeper kafka-zookeeper:2181 --topic flights-realtime --create --partitions 1 --replication-factor 1 --if-not-exists"
	arguments := strings.Split(cmd," ")
	err = kubelib.RunKubectlWithKubeConfig(commandEnvironment.Get(framework.CmdEnvKubectlExecutable), kubeConfig, arguments)
	if err != nil {
		return err
	}

	cmd = "-n pinot-quickstart exec kafka-0 -- kafka-topics --zookeeper kafka-zookeeper:2181 --topic flights-realtime-avro --create --partitions 1 --replication-factor 1 --if-not-exists"
	arguments = strings.Split(cmd," ")
	err = kubelib.RunKubectlWithKubeConfig(commandEnvironment.Get(framework.CmdEnvKubectlExecutable), kubeConfig, arguments)
	if err != nil {
		return err
	}

	return nil
}

func CreatePinotRealtimeIngestion(commandEnvironment framework.CommandEnvironment, region string, eksClusterName string) error {
	kubeConfig, err := awslib.CreateKubeConfig(region, commandEnvironment.Get(framework.CmdEnvKubeConfig), eksClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %s", err.Error())
	}
	defer kubeConfig.Cleanup()
	cmd := fmt.Sprintf("apply -f %s/pinot-realtime-quickstart.yml",
		commandEnvironment.Get(CmdEnvPinotHelmChart))
	arguments := strings.Split(cmd," ")
	err = kubelib.RunKubectlWithKubeConfig(commandEnvironment.Get(framework.CmdEnvKubectlExecutable), kubeConfig, arguments)
	if err != nil {
		return err
	}
	return nil
}