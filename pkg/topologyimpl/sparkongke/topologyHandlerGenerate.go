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
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/gke"
	"github.com/datapunchorg/punch/pkg/topologyimpl/sparkoneks"
)

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	namePrefix := framework.DefaultNamePrefixTemplate

	topologyName := fmt.Sprintf("%s-sparkongke-01", namePrefix)
	gkeClusterName := topologyName

	gkeTopologyHandler := gke.TopologyHandler{}
	gkeGeneratedTopology, err := gkeTopologyHandler.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Gke topology for %s: %w", KindSparkOnGkeTopology, err)
	}
	gkeTopology := gkeGeneratedTopology.(*gke.Topology)

	sparkOnEksTopologyHandler := sparkoneks.TopologyHandler{}
	sparkOnEksGeneratedTopology, err := sparkOnEksTopologyHandler.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SparkOnEks topology for %s: %w", KindSparkOnGkeTopology, err)
	}
	sparkOnEksTopology := sparkOnEksGeneratedTopology.(*sparkoneks.SparkOnEksTopology)

	topology := Topology{
		TopologyBase: framework.TopologyBase{
			ApiVersion: framework.DefaultVersion,
			Kind:       KindSparkOnGkeTopology,
			Metadata: framework.TopologyMetadata{
				Name:               topologyName,
				CommandEnvironment: map[string]string{},
				Notes:              map[string]string{},
			},
		},
		Spec: TopologySpec{
			GkeSpec:               gkeTopology.Spec,
			ArgocdInstallYamlFile: "https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml",
			Spark:                 sparkOnEksTopology.Spec.Spark,
		},
	}

	for k, v := range sparkOnEksTopology.Metadata.CommandEnvironment {
		topology.Metadata.CommandEnvironment[k] = v
	}

	for k, v := range gkeTopology.Metadata.CommandEnvironment {
		topology.Metadata.CommandEnvironment[k] = v
	}

	topology.Spec.GkeSpec.GkeCluster.ClusterName = gkeClusterName

	return &topology, nil
}
