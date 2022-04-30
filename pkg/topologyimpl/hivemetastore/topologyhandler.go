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
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/kubelib"
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"gopkg.in/yaml.v3"
	"regexp"
	"strings"
)

var nonAlphanumericRegexp *regexp.Regexp

func init() {
	framework.DefaultTopologyHandlerManager.AddHandler(KindHiveMetastoreTopology, &TopologyHandler{})

	var err error
	nonAlphanumericRegexp, err = regexp.Compile("[^a-zA-Z]+")
	if err != nil {
		panic(err)
	}
}

type TopologyHandler struct {
}

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	topology := GenerateHiveMetastoreTopology()
	return &topology, nil
}

func (t *TopologyHandler) Parse(yamlContent []byte) (framework.Topology, error) {
	result := CreateDefaultHiveMetastoreTopology(DefaultNamePrefix, eks.ToBeReplacedS3BucketName)
	err := yaml.Unmarshal(yamlContent, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML (%s): \n%s", err.Error(), string(yamlContent))
	}
	return &result, nil
}

func (t *TopologyHandler) Validate(topology framework.Topology, phase string) (framework.Topology, error) {
	specificTopology := topology.(*HiveMetastoreTopology)

	if strings.EqualFold(phase, framework.PhaseBeforeInstall) {
		if specificTopology.Spec.Database.UseExternalDb {
			if specificTopology.Spec.Database.UserPassword == "" || specificTopology.Spec.Database.UserPassword == framework.TemplateNoValue {
				return nil, fmt.Errorf("spec.dbUserPassword is emmpty, please provide the value for the password")
			}
		}
	}

	return topology, nil
}

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	specificTopology := topology.(*HiveMetastoreTopology)
	commandEnvironment := framework.CreateCommandEnvironment(specificTopology.Metadata.CommandEnvironment)
	if commandEnvironment.GetBoolOrElse(CmdEnvWithMinikube, false) {
		commandEnvironment.Set(CmdEnvKubeConfig, kubelib.GetKubeConfigPath())
	}
	deployment, err := eks.BuildInstallDeployment(specificTopology.Spec.EksSpec, commandEnvironment)
	if err != nil {
		return nil, err
	}
	if !specificTopology.Spec.Database.UseExternalDb {
		deployment.AddStep("createHiveMetastoreDatabase", "Create Hive Metastore database", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
			databaseInfo, err := CreatePostgresqlDatabase(commandEnvironment, specificTopology.Spec)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
			return framework.DeploymentStepOutput{"databaseInfo": databaseInfo}, nil
		})
	}
	deployment.AddStep("initHiveMetastoreDatabase", "Init Hive Metastore database", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
		var databaseInfo DatabaseInfo
		if !specificTopology.Spec.Database.UseExternalDb {
			databaseInfo = c.GetStepOutput("createHiveMetastoreDatabase")["databaseInfo"].(DatabaseInfo)
		} else {
			databaseInfo = DatabaseInfo{
				ConnectionString: specificTopology.Spec.Database.ConnectionString,
				UserName: specificTopology.Spec.Database.UserName,
				UserPassword: specificTopology.Spec.Database.UserPassword,
			}
		}
		err := InitDatabase(commandEnvironment, specificTopology.Spec, databaseInfo)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		return framework.NewDeploymentStepOutput(), nil
	})
	deployment.AddStep("installHiveMetastoreServer", "Install Hive Metastore server", func(c framework.DeploymentContext) (framework.DeploymentStepOutput, error) {
		var databaseInfo DatabaseInfo
		if !specificTopology.Spec.Database.UseExternalDb {
			databaseInfo = c.GetStepOutput("createHiveMetastoreDatabase")["databaseInfo"].(DatabaseInfo)
		} else {
			databaseInfo = DatabaseInfo{
				ConnectionString: specificTopology.Spec.Database.ConnectionString,
				UserName: specificTopology.Spec.Database.UserName,
				UserPassword: specificTopology.Spec.Database.UserPassword,
			}
		}
		err := InstallMetastoreServer(commandEnvironment, specificTopology.Spec, databaseInfo)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		return framework.DeploymentStepOutput{"TODO": ""}, nil
	})
	err = deployment.Run()
	return deployment.GetOutput(), err
}

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	specificTopology := topology.(*HiveMetastoreTopology)
	commandEnvironment := framework.CreateCommandEnvironment(specificTopology.Metadata.CommandEnvironment)
	deployment, err := eks.BuildUninstallDeployment(specificTopology.Spec.EksSpec, commandEnvironment)
	if err != nil {
		return nil, err
	}
	return deployment.GetOutput(), nil
}
