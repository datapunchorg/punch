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

package argocdongke

import (
	"fmt"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/topologyimpl/gke"
)

func (t *TopologyHandler) Generate() (framework.Topology, error) {
	namePrefix := framework.DefaultNamePrefixTemplate

	topologyName := fmt.Sprintf("%s-argocdongke-01", namePrefix)
	gkeClusterName := topologyName

	gkeTopologyHandler := gke.TopologyHandler{}
	gkeGeneratedTopology, err := gkeTopologyHandler.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Gke topology for ArgocdOnGke: %w", err)
	}
	gkeTopology := gkeGeneratedTopology.(*gke.Topology)

	topology := Topology{
		TopologyBase: framework.TopologyBase{
			ApiVersion: framework.DefaultVersion,
			Kind:       KindArgocdOnGkeTopology,
			Metadata:   gkeTopology.Metadata,
		},
		Spec: TopologySpec{
			TopologySpec: gkeTopology.Spec,
		},
	}

	topology.Spec.GkeCluster.ClusterName = gkeClusterName

	return &topology, nil
}
