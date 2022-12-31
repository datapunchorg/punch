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
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/gke"
	"github.com/datapunchorg/punch/pkg/topologyimpl/sparkoneks"
)

const (
	DefaultNginxIngressHelmInstallName = "ingress-nginx"
	DefaultNginxIngressNamespace       = "ingress-nginx"
	DefaultNginxEnableHttp             = true
	DefaultNginxEnableHttps            = true

	KindSparkOnGkeTopology = "SparkOnGke"

	CmdEnvNginxHelmChart             = "nginxHelmChart"
	CmdEnvClusterAutoscalerHelmChart = "ClusterAutoscalerHelmChart"

	DefaultZone = "us-central1-c"
)

type Topology struct {
	framework.TopologyBase `json:",inline" yaml:",inline"`
	Spec                   TopologySpec `json:"spec" yaml:"spec"`
}

type TopologySpec struct {
	GkeSpec           gke.TopologySpec              `json:",inline" yaml:",inline"`
	ArgocdInstallYaml string                        `json:"argocdInstallYaml" yaml:"argocdInstallYaml"`
	Spark             sparkoneks.SparkComponentSpec `json:"spark" yaml:"spark"`
}

func (t *Topology) GetKind() string {
	return t.Kind
}

func (t *Topology) GetMetadata() *framework.TopologyMetadata {
	return &t.Metadata
}

// https://cloud.google.com/docs/authentication/application-default-credentials
// gcloud auth application-default login
// or: export GOOGLE_APPLICATION_CREDENTIALS=/foo/credential.json

type TopologyHandler struct {
}
