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
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/resource"
)

func (t *TopologyHandler) Uninstall(topology framework.Topology) (framework.DeploymentOutput, error) {
	currentTopology := topology.(*Topology)

	commandEnvironment := framework.CreateCommandEnvironment(currentTopology.Metadata.CommandEnvironment)

	deployment, err := CreateUninstallDeployment(currentTopology.Spec, commandEnvironment)
	if err != nil {
		return deployment.GetOutput(), err
	}

	err = deployment.Run()
	return deployment.GetOutput(), err
}

/*
func (t *TopologyHandler) PrintNotes(topology framework.Topology, deploymentOutput framework.DeploymentOutput) {
	currentTopology := topology.(*Topology)

	str := `
------------------------------
Example commands to use EKS cluster:
------------------------------
Step 1: run: aws eks update-kubeconfig --region %s --name %s
Step 2: run: kubectl get pods -A`
	log.Printf(str, currentTopology.Spec.Region, currentTopology.Spec.EksCluster.ClusterName)
}*/

func CreateUninstallDeployment(topologySpec TopologySpec, commandEnvironment framework.CommandEnvironment) (framework.Deployment, error) {
	deployment := framework.NewDeployment()

	/*deployment.AddStep("deleteLoadBalancers", "Delete Load Balancers in EKS Cluster", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		err := awslib.DeleteAllLoadBalancersOnEks(topologySpec.Region, topologySpec.VpcId, topologySpec.EksCluster.ClusterName)
		return framework.NewDeploymentStepOutput(), err
	})
	deployment.AddStep("deleteOidcProvider", "Delete OIDC Provider", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		clusterSummary, err := resource.DescribeEksCluster(topologySpec.Region, topologySpec.EksCluster.ClusterName)
		if err != nil {
			log.Printf("[WARN] Cannot delete OIDC provider, failed to get EKS cluster %s in regsion %s: %s", topologySpec.EksCluster.ClusterName, topologySpec.Region, err.Error())
			return framework.NewDeploymentStepOutput(), nil
		}
		if clusterSummary.OidcIssuer != "" {
			log.Printf("Deleting OIDC Identity Provider %s", clusterSummary.OidcIssuer)
			err = awslib.DeleteOidcProvider(topologySpec.Region, clusterSummary.OidcIssuer)
			if err != nil {
				log.Printf("[WARN] Failed to delete OIDC provider %s: %s", clusterSummary.OidcIssuer, err.Error())
				return framework.NewDeploymentStepOutput(), nil
			}
			log.Printf("Deleted OIDC Identity Provider %s", clusterSummary.OidcIssuer)
		}
		return framework.NewDeploymentStepOutput(), nil
	})
	deployment.AddStep("deleteNodeGroups", "Delete Node Groups", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		for _, nodeGroup := range topologySpec.NodeGroups {
			err := awslib.DeleteNodeGroup(topologySpec.Region, topologySpec.EksCluster.ClusterName, nodeGroup.Name)
			if err != nil {
				return framework.NewDeploymentStepOutput(), err
			}
		}
		return framework.NewDeploymentStepOutput(), nil
	}) */
	deployment.AddStep("deleteGkeCluster", "Delete GKE Cluster", func(c framework.DeploymentContext) (framework.DeployableOutput, error) {
		resource.DeleteGkeCluster(topologySpec.ProjectId, topologySpec.Location, topologySpec.GkeCluster.ClusterName)
		return framework.NewDeploymentStepOutput(), nil
	})

	return deployment, nil
}
