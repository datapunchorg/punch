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

package hivemetastore

import (
	"context"
	"fmt"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DatabaseInfo struct {
	Host string
	Port int
	UserName string
	UserPassword string
}

// TODO mask DatabaseInfo.UserPassword in YAML output

func (t DatabaseInfo) String() string {
	copy := t
	copy.UserPassword = FieldMaskValue
	bytes, err := yaml.Marshal(copy)
	if err != nil {
		return fmt.Sprintf("(failed to get string for DatabaseInfo: %s)", err.Error())
	}
	return string(bytes)
}

func CreatePostgresqlDatabase(commandEnvironment framework.CommandEnvironment, spec HiveMetastoreTopologySpec) (DatabaseInfo, error) {
	kubeConfig, err := awslib.CreateKubeConfig(spec.EksSpec.Region, commandEnvironment.Get(eks.CmdEnvKubeConfig), spec.EksSpec.Eks.ClusterName)
	if err != nil {
		return DatabaseInfo{}, fmt.Errorf("failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := "postgresql"
	namespace := spec.Namespace

	kubelib.RunHelm(commandEnvironment.Get(eks.CmdEnvHelmExecutable),[]string{"repo", "add", "bitnami", "https://charts.bitnami.com/bitnami"})
	kubelib.RunHelm(commandEnvironment.Get(eks.CmdEnvHelmExecutable),[]string{"search", "repo", "postgres"})

	arguments := []string{}
	kubelib.InstallHelm(commandEnvironment.Get(eks.CmdEnvHelmExecutable), "bitnami/postgresql", kubeConfig, arguments, installName, namespace)

	_, clientset, err := awslib.CreateKubernetesClient(spec.EksSpec.Region, commandEnvironment.Get(CmdEnvKubeConfig), spec.EksSpec.Eks.ClusterName)
	if err != nil {
		return DatabaseInfo{}, fmt.Errorf("failed to create Kubernetes client: %s", err.Error())
	}

	podNamePrefix := "postgresql"
	err = kubelib.WaitPodsInPhase(clientset, namespace, podNamePrefix, v1.PodRunning)
	if err != nil {
		return DatabaseInfo{}, fmt.Errorf("pod %s*** in namespace %s is not in phase %s", podNamePrefix, namespace, v1.PodRunning)
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
	kubelib.InstallHelm(commandEnvironment.Get(eks.CmdEnvHelmExecutable), hiveMetastoreCreateDatabaseHelmChart, kubeConfig, arguments, installName, namespace)

	return DatabaseInfo{
		Host: dbServerHost,
		Port: 5432,
		UserName: "postgres",
		// TODO mask UserPassword in logging
		UserPassword: string(password),
	}, nil
}

func InitDatabase(commandEnvironment framework.CommandEnvironment, spec HiveMetastoreTopologySpec, databaseInfo DatabaseInfo) error {
	kubeConfig, err := awslib.CreateKubeConfig(spec.EksSpec.Region, commandEnvironment.Get(eks.CmdEnvKubeConfig), spec.EksSpec.Eks.ClusterName)
	if err != nil {
		return fmt.Errorf("Failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := "hive-metastore-init-db"
	namespace := spec.Namespace

	arguments := []string{
		"--set", fmt.Sprintf("image.name=%s", spec.ImageRepository),
		"--set", fmt.Sprintf("image.tag=%s", spec.ImageTag),
		"--set", fmt.Sprintf("dbServerHost=%s", databaseInfo.Host),
		"--set", fmt.Sprintf("dbServerPort=%d", databaseInfo.Port),
		"--set", fmt.Sprintf("dbUserName=%s", databaseInfo.UserName),
		"--set", fmt.Sprintf("dbUserPassword=%s", databaseInfo.UserPassword),
	}

	kubelib.InstallHelm(commandEnvironment.Get(eks.CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvHiveMetastoreInitHelmChart), kubeConfig, arguments, installName, namespace)

	return nil
}