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
	"github.com/datapunchorg/punch/pkg/topologyimpl/eks"
	"gopkg.in/yaml.v3"
)

const (
	KindHiveMetastoreTopology = "HiveMetastore"

	FieldMaskValue = "***"

	CmdEnvPostgresqlHelmChart = "postgresqlHelmChart"
	CmdEnvHiveMetastoreCreateDatabaseHelmChart = "hiveMetastoreCreateDatabaseHelmChart"
	CmdEnvHiveMetastoreInitHelmChart = "hiveMetastoreInitHelmChart"
	CmdEnvHiveMetastoreServerHelmChart = "hiveMetastoreServerHelmChart"
)

type HiveMetastoreTopology struct {
	framework.TopologyBase               `json:",inline" yaml:",inline"`
	Spec       HiveMetastoreTopologySpec      `json:"spec"`
}

type HiveMetastoreTopologySpec struct {
	NamePrefix        string                   `json:"namePrefix" yaml:"namePrefix"`
	Region            string                   `json:"region" yaml:"region"`
	EksClusterName    string                   `json:"eksClusterName" yaml:"eksClusterName"`
	EksVpcId          string                          `json:"eksVpcId" yaml:"eksVpcId"`
    Namespace        string `json:"namespace" yaml:"namespace"`
	ImageRepository  string `json:"imageRepository" yaml:"imageRepository"`
	ImageTag string `json:"imageTag" yaml:"imageTag"`
	Database  HiveMetastoreDatabaseSpec `json:"database" yaml:"database"`
	WarehouseDir string `json:"warehouseDir" yaml:"warehouseDir"`
}

type HiveMetastoreDatabaseSpec struct {
	ExternalDb       bool   `json:"useExternalDb" yaml:"useExternalDb"`
	ConnectionString string `json:"connectionString" yaml:"connectionString"`
	User       string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`
}

func GenerateHiveMetastoreTopology() HiveMetastoreTopology {
	namePrefix := framework.DefaultNamePrefixTemplate
	s3BucketName := framework.DefaultS3BucketNameTemplate
	return CreateDefaultHiveMetastoreTopology(namePrefix, s3BucketName)
}

func CreateDefaultHiveMetastoreTopology(namePrefix string, s3BucketName string) HiveMetastoreTopology {
	topologyName := fmt.Sprintf("%s-db-01", namePrefix)

	eksTopology := eks.CreateDefaultEksTopology(namePrefix, s3BucketName)

	topology := HiveMetastoreTopology{
		TopologyBase: framework.TopologyBase{
			ApiVersion: framework.DefaultVersion,
			Kind: KindHiveMetastoreTopology,
			Metadata: framework.TopologyMetadata{
				Name:               topologyName,
				CommandEnvironment: map[string]string{
					framework.CmdEnvHelmExecutable: framework.DefaultHelmExecutable,
					CmdEnvPostgresqlHelmChart: "third-party/helm-charts/bitnami/charts/postgresql",
					CmdEnvHiveMetastoreCreateDatabaseHelmChart: "third-party/helm-charts/hive-metastore/charts/hive-metastore-postgresql-create-db",
					CmdEnvHiveMetastoreInitHelmChart: "third-party/helm-charts/hive-metastore/charts/hive-metastore-init-postgresql",
					CmdEnvHiveMetastoreServerHelmChart: "third-party/helm-charts/hive-metastore/charts/hive-metastore-server",
				},
				Notes:              map[string]string{},
			},
		},
		Spec: HiveMetastoreTopologySpec{
			NamePrefix:   namePrefix,
			Region:       fmt.Sprintf("{{ or .Values.region `%s` }}", framework.DefaultRegion),
			EksClusterName:                eksTopology.Spec.Eks.ClusterName,
			EksVpcId:                      eksTopology.Spec.VpcId,
			Namespace: "hive-01",
			ImageRepository: "ghcr.io/datapunchorg/helm-hive-metastore",
			ImageTag: "main-1650942144",
			Database: HiveMetastoreDatabaseSpec{
				ExternalDb:       false,
				ConnectionString: "",
				User:             "",
				Password:         "",
			},
			WarehouseDir: fmt.Sprintf("s3a://%s/punch/%s/warehouse", s3BucketName, namePrefix),
		},
	}

	return topology
}

func (t *HiveMetastoreTopology) GetKind() string {
	return t.Kind
}

func (t *HiveMetastoreTopology) GetMetadata() *framework.TopologyMetadata {
	return &t.Metadata
}

func (t *HiveMetastoreTopology) String() string {
	topologyBytes, err := yaml.Marshal(t)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology: %s)", err.Error())
	}
	var copy HiveMetastoreTopology
	err = yaml.Unmarshal(topologyBytes, &copy)
	if err != nil {
		return fmt.Sprintf("(Failed to deserialize topology in ToYamlString(): %s)", err.Error())
	}
	if copy.Spec.Database.Password != "" {
		copy.Spec.Database.Password = FieldMaskValue
	}
	topologyBytes, err = yaml.Marshal(copy)
	if err != nil {
		return fmt.Sprintf("(Failed to serialize topology in String(): %s)", err.Error())
	}
	return string(topologyBytes)
}
