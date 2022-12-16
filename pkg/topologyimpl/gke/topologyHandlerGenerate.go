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

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/resource"
)

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	namePrefix := framework.DefaultNamePrefixTemplate

	topologyName := fmt.Sprintf("%s-gke-01", namePrefix)
	gkeClusterName := topologyName
	topology := Topology{
		TopologyBase: framework.TopologyBase{
			ApiVersion: framework.DefaultVersion,
			Kind:       KindGkeTopology,
			Metadata: framework.TopologyMetadata{
				Name: topologyName,
				CommandEnvironment: map[string]string{
					framework.CmdEnvHelmExecutable:    framework.DefaultHelmExecutable,
					framework.CmdEnvKubectlExecutable: framework.DefaultKubectlExecutable,
					framework.CmdEnvWithMinikube:      "false",
					CmdEnvNginxHelmChart:              "third-party/helm-charts/ingress-nginx/charts/ingress-nginx",
					CmdEnvClusterAutoscalerHelmChart:  "third-party/helm-charts/cluster-autoscaler/charts/cluster-autoscaler",
				},
				Notes: map[string]string{},
			},
		},
		Spec: TopologySpec{
			NamePrefix: namePrefix,
			ProjectId:  fmt.Sprintf("{{ .Values.projectId }}"),
			Location:   fmt.Sprintf("{{ or .Values.location `%s` }}", DefaultZone),
			GkeCluster: resource.GkeCluster{
				ClusterName:      gkeClusterName,
				InitialNodeCount: 2,
			},
			NginxIngress: NginxIngress{
				HelmInstallName: DefaultNginxIngressHelmInstallName,
				Namespace:       DefaultNginxIngressNamespace,
				EnableHttp:      DefaultNginxEnableHttp,
				EnableHttps:     DefaultNginxEnableHttps,
			},
		},
	}
	return &topology, nil
}
