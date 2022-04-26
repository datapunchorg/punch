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
	kubeConfig, err := awslib.CreateKubeConfig(spec.Eks.Region, commandEnvironment.Get(eks.CmdEnvKubeConfig), spec.Eks.Eks.ClusterName)
	if err != nil {
		log.Fatalf("Failed to get kube config: %s", err)
	}

	defer kubeConfig.Cleanup()

	installName := spec.HelmInstallName
	namespace := spec.Namespace

	arguments := []string{
		"--set", fmt.Sprintf("image.name=%s", spec.ImageRepository),
		"--set", fmt.Sprintf("image.tag=%s", spec.ImageTag),
	}

	kubelib.InstallHelm(commandEnvironment.Get(eks.CmdEnvHelmExecutable), commandEnvironment.Get(HiveMetastoreInitHelmChart), kubeConfig, arguments, installName, namespace)
}