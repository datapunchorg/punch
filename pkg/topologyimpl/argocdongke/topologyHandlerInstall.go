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
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/resource"
	"github.com/datapunchorg/punch/pkg/topologyimpl/gke"
)

func (t *TopologyHandler) Install(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*Topology)

	deployment := framework.NewDeployment()

	deployment.AddStep("createGkeCluster", "Create GKE cluster", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		err := resource.CreateGkeCluster(currentTopology.Spec.GkeSpec.ProjectId, currentTopology.Spec.GkeSpec.Location, currentTopology.Spec.GkeSpec.GkeCluster)
		if err != nil {
			return framework.NewDeploymentStepOutput(), err
		}
		return framework.DeployableOutput{}, nil
	})

	if currentTopology.GetMetadata().GetCommandEnvironment().Get(CmdEnvNginxHelmChart) != "" {
		deployment.AddStep("deployNginxIngressController", "Deploy Nginx ingress controller", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
			return gke.DeployNginxIngressController(*currentTopology.GetMetadata().GetCommandEnvironment(), currentTopology.Spec.GkeSpec)
		})
	}

	err := deployment.Run()
	return deployment.GetOutput(), err
}
