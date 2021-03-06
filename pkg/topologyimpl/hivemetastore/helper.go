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

package hivemetastore

import (
	"context"
	"fmt"
	"github.com/datapunchorg/punch/pkg/common"

	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var NODE_PORT_LOCAL_THRIFT_PORT = 30083
var WAREHOUSE_DIR_LOCAL_FILE_TEMP_DIRECTORY = "file:///tmp/"

type DatabaseInfo struct {
	ConnectionString string
	User             string
	Password         string
}

// TODO mask DatabaseInfo.Password in YAML output

func (t DatabaseInfo) String() string {
	copy := t
	copy.Password = framework.FieldMaskValue
	bytes, err := yaml.Marshal(copy)
	if err != nil {
		return fmt.Sprintf("(failed to get string for DatabaseInfo: %s)", err.Error())
	}
	return string(bytes)
}

func CreatePostgresqlDatabase(commandEnvironment framework.CommandEnvironment, spec HiveMetastoreTopologySpec) (DatabaseInfo, error) {
	kubeConfig, err := awslib.CreateKubeConfig(spec.Region, commandEnvironment.Get(framework.CmdEnvKubeConfig), spec.EksClusterName)
	if err != nil {
		return DatabaseInfo{}, fmt.Errorf("failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := "postgresql"
	namespace := spec.Namespace

	//kubelib.RunHelm(commandEnvironment.Get(eks.CmdEnvHelmExecutable),[]string{"repo", "add", "bitnami", "https://charts.bitnami.com/bitnami"})
	//kubelib.RunHelm(commandEnvironment.Get(eks.CmdEnvHelmExecutable),[]string{"search", "repo", "postgres"})
	//kubelib.InstallHelm(commandEnvironment.Get(eks.CmdEnvHelmExecutable), "bitnami/postgresql", kubeConfig, arguments, installName, namespace)

	arguments := []string{}
	postgresqlHelmChart := commandEnvironment.Get(CmdEnvPostgresqlHelmChart)
	kubelib.InstallHelm(commandEnvironment.Get(framework.CmdEnvHelmExecutable), postgresqlHelmChart, kubeConfig, arguments, installName, namespace)

	_, clientset, err := awslib.CreateKubernetesClient(spec.Region, commandEnvironment.Get(framework.CmdEnvKubeConfig), spec.EksClusterName)
	if err != nil {
		return DatabaseInfo{}, fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	podNamePrefix := "postgresql"
	waitPosStatus := []v1.PodPhase{v1.PodRunning}
	err = kubelib.WaitPodsInPhases(clientset, namespace, podNamePrefix, waitPosStatus)
	if err != nil {
		return DatabaseInfo{}, fmt.Errorf("pod %s*** in namespace %s is not in phase %s", podNamePrefix, namespace, waitPosStatus)
	}

	secretName := "postgresql"
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return DatabaseInfo{}, fmt.Errorf("failed to get secret %s in namespace %s", secretName, namespace)
	}
	password, ok := secret.Data["postgres-password"]
	if !ok {
		return DatabaseInfo{}, fmt.Errorf("failed to find postgres password from secret %s in namespace %s: %s", secretName, namespace, err.Error())
	}

	installName = "hive-metastore-postgresql-create-db"
	dbServerHost := fmt.Sprintf("postgresql.%s.svc.cluster.local", namespace)
	arguments = []string{
		"--set",
		fmt.Sprintf("dbServerHost=%s", dbServerHost),
		"--set",
		fmt.Sprintf("dbUserPassword=%s", password),
	}
	hiveMetastoreCreateDatabaseHelmChart := commandEnvironment.Get(CmdEnvHiveMetastoreCreateDatabaseHelmChart)
	kubelib.InstallHelm(commandEnvironment.Get(framework.CmdEnvHelmExecutable), hiveMetastoreCreateDatabaseHelmChart, kubeConfig, arguments, installName, namespace)

	podNamePrefix = "hive-metastore-postgresql-create-db"
	waitPosStatus = []v1.PodPhase{v1.PodSucceeded}
	err = kubelib.WaitPodsInPhases(clientset, namespace, podNamePrefix, waitPosStatus)
	if err != nil {
		return DatabaseInfo{}, fmt.Errorf("pod %s*** in namespace %s is not in phase %s", podNamePrefix, namespace, waitPosStatus)
	}

	return DatabaseInfo{
		ConnectionString: fmt.Sprintf("jdbc:postgresql://%s:5432/metastore_db", dbServerHost),
		User:             "postgres",
		// TODO mask Password in logging
		Password: string(password),
	}, nil
}

func InitDatabase(commandEnvironment framework.CommandEnvironment, spec HiveMetastoreTopologySpec, databaseInfo DatabaseInfo) error {
	kubeConfig, err := awslib.CreateKubeConfig(spec.Region, commandEnvironment.Get(framework.CmdEnvKubeConfig), spec.EksClusterName)
	if err != nil {
		return fmt.Errorf("Failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := "hive-metastore-init-db"
	namespace := spec.Namespace

	arguments := []string{
		"--set", fmt.Sprintf("image.name=%s", spec.ImageRepository),
		"--set", fmt.Sprintf("image.tag=%s", spec.ImageTag),
		"--set", fmt.Sprintf("dbConnectionString=%s", databaseInfo.ConnectionString),
		"--set", fmt.Sprintf("dbUserName=%s", databaseInfo.User),
		"--set", fmt.Sprintf("dbUserPassword=%s", databaseInfo.Password),
	}

	kubelib.InstallHelm(commandEnvironment.Get(framework.CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvHiveMetastoreInitHelmChart), kubeConfig, arguments, installName, namespace)

	_, clientset, err := awslib.CreateKubernetesClient(spec.Region, commandEnvironment.Get(framework.CmdEnvKubeConfig), spec.EksClusterName)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	podNamePrefix := "hive-metastore-init-postgresql"
	waitPosStatus := []v1.PodPhase{v1.PodSucceeded, v1.PodFailed} // TODO deal with repeated init postgresql and do not wait PodFailed
	err = kubelib.WaitPodsInPhases(clientset, namespace, podNamePrefix, waitPosStatus)
	if err != nil {
		return fmt.Errorf("pod %s*** in namespace %s is not in phase %s", podNamePrefix, namespace, waitPosStatus)
	}

	return nil
}

func InstallMetastoreServer(commandEnvironment framework.CommandEnvironment, spec HiveMetastoreTopologySpec, databaseInfo DatabaseInfo) ([]string, error) {
	kubeConfig, err := awslib.CreateKubeConfig(spec.Region, commandEnvironment.Get(framework.CmdEnvKubeConfig), spec.EksClusterName)
	if err != nil {
		return nil, fmt.Errorf("Failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := "hive-metastore-server"
	namespace := spec.Namespace

	arguments := []string{
		"--set", fmt.Sprintf("image.name=%s", spec.ImageRepository),
		"--set", fmt.Sprintf("image.tag=%s", spec.ImageTag),
		"--set", fmt.Sprintf("dbConnectionString=%s", databaseInfo.ConnectionString),
		"--set", fmt.Sprintf("dbUserName=%s", databaseInfo.User),
		"--set", fmt.Sprintf("dbUserPassword=%s", databaseInfo.Password),
	}

	if !commandEnvironment.GetBoolOrElse(framework.CmdEnvWithMinikube, false) {
		arguments = append(arguments, "--set")
		arguments = append(arguments, fmt.Sprintf("metastoreWarehouseDir=%s", spec.WarehouseDir))
	} else {
		arguments = append(arguments, "--set")
		arguments = append(arguments, fmt.Sprintf("metastoreWarehouseDir=%s", WAREHOUSE_DIR_LOCAL_FILE_TEMP_DIRECTORY))
		arguments = append(arguments, "--set", "service.type=NodePort")
		arguments = append(arguments, "--set", fmt.Sprintf("service.nodePorts.thrift=%d", NODE_PORT_LOCAL_THRIFT_PORT))
	}

	kubelib.InstallHelm(commandEnvironment.Get(framework.CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvHiveMetastoreServerHelmChart), kubeConfig, arguments, installName, namespace)

	_, clientset, err := awslib.CreateKubernetesClient(spec.Region, commandEnvironment.Get(framework.CmdEnvKubeConfig), spec.EksClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	podNamePrefix := "hive-metastore-server"
	waitPosStatus := []v1.PodPhase{v1.PodRunning}
	err = kubelib.WaitPodsInPhases(clientset, namespace, podNamePrefix, waitPosStatus)
	if err != nil {
		return nil, fmt.Errorf("pod %s*** in namespace %s is not in phase %s", podNamePrefix, namespace, waitPosStatus)
	}

	serviceName := "hive-metastore"
	waitLoadBalancerReady := !commandEnvironment.GetBoolOrElse(framework.CmdEnvWithMinikube, false)
	var hostPorts []common.HostPort
	if waitLoadBalancerReady {
		hostPorts, err = awslib.WaitServiceLoadBalancerHostPorts(spec.Region, commandEnvironment.Get(framework.CmdEnvKubeConfig), spec.EksClusterName, namespace, serviceName)
		if err != nil {
			return nil, err
		}
	} else {
		hostPorts, err = awslib.GetServiceLoadBalancerHostPorts(spec.Region, commandEnvironment.Get(framework.CmdEnvKubeConfig), spec.EksClusterName, namespace, serviceName)
		if err != nil {
			return nil, err
		}
	}
	urls := make([]string, 0, len(hostPorts))
	for _, entry := range hostPorts {
		urls = append(urls, fmt.Sprintf("thrift://%s:%d", entry.Host, entry.Port))
	}
	return urls, nil
}
