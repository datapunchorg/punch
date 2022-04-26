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
	"fmt"
	"github.com/datapunchorg/punch/pkg/awslib"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"log"
)

func InitDatabase(commandEnvironment framework.CommandEnvironment, spec HiveMetastoreTopologySpec) () {
	kubeConfig, err := awslib.CreateKubeConfig(spec.Eks.Region, commandEnvironment.Get(eks.CmdEnvKubeConfig), topology.EksSpec.Eks.ClusterName)
	if err != nil {
		log.Fatalf("Failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := topology.SparkOperator.HelmInstallName
	operatorNamespace := topology.SparkOperator.Namespace
	sparkApplicationNamespace := topology.SparkOperator.SparkApplicationNamespace

	arguments := []string{
		"--set", fmt.Sprintf("sparkJobNamespace=%s", sparkApplicationNamespace),
		"--set", fmt.Sprintf("image.repository=%s", topology.SparkOperator.ImageRepository),
		"--set", fmt.Sprintf("image.tag=%s", topology.SparkOperator.ImageTag),
		"--set", "serviceAccounts.spark.create=true",
		"--set", "serviceAccounts.spark.name=spark",
		"--set", "apiGateway.userName=" + topology.ApiGateway.UserName,
		"--set", "apiGateway.userPassword=" + topology.ApiGateway.UserPassword,
		"--set", "apiGateway.s3Region=" + topology.EksSpec.Region,
		"--set", "apiGateway.s3Bucket=" + topology.EksSpec.S3BucketName,
		// "--set", "webhook.enable=true",
	}

	if !commandEnvironment.GetBoolOrElse(eks.CmdEnvWithMinikube, false) {
		arguments = append(arguments, "--set")
		arguments = append(arguments, "apiGateway.sparkEventLogEnabled=true")
		arguments = append(arguments, "--set")
		arguments = append(arguments, "apiGateway.sparkEventLogDir=" + topology.ApiGateway.SparkEventLogDir)
	}

	kubelib.InstallHelm(commandEnvironment.Get(eks.CmdEnvHelmExecutable), commandEnvironment.Get(CmdEnvSparkOperatorHelmChart), kubeConfig, arguments, installName, operatorNamespace)
}