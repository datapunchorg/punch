package resource

import (
	"context"
	"fmt"
	"github.com/datapunchorg/punch/pkg/common"
	"google.golang.org/api/container/v1"
	"log"
)

type GkeCluster struct {
	ClusterName string `json:"clusterName" yaml:"clusterName"`
}

func CreateGkeCluster(projectId string, zone string, gkeCluster GkeCluster) error {
	ctx := context.Background()
	containerService, err := container.NewService(ctx)
	if err != nil {
		return fmt.Errorf("failed to new container service client: %w", err)
	}
	createClusterRequest := &container.CreateClusterRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, zone),
		Cluster: &container.Cluster{
			Name:             gkeCluster.ClusterName,
			InitialNodeCount: 1,
		},
	}
	projectsZonesClustersCreateCall := containerService.Projects.Zones.Clusters.Create("", "", createClusterRequest)
	operation, err := projectsZonesClustersCreateCall.Do()
	if err != nil {
		return fmt.Errorf("failed to run %s: %w", common.GetReflectTypeName(projectsZonesClustersCreateCall), err)
	}
	log.Printf("Finished operation %s", operation.Name)

	projectsZonesClustersGetCall := containerService.Projects.Zones.Clusters.Get(projectId, zone, gkeCluster.ClusterName)
	getClusterResult, err := projectsZonesClustersGetCall.Do()
	if err != nil {
		return fmt.Errorf("failed to run %s: %w", common.GetReflectTypeName(projectsZonesClustersGetCall), err)
	}
	log.Printf("Cluster %s (id: %s) status: %s", gkeCluster.ClusterName, getClusterResult.Id, getClusterResult.Status)

	return nil
}

func DeleteGkeCluster(projectId string, zone string, clusterId string) error {
	ctx := context.Background()
	containerService, err := container.NewService(ctx)
	if err != nil {
		return fmt.Errorf("failed to new container service client: %w", err)
	}
	projectsZonesClustersDeleteCall := containerService.Projects.Zones.Clusters.Delete(projectId, zone, clusterId)
	operation, err := projectsZonesClustersDeleteCall.Do()
	if err != nil {
		return fmt.Errorf("failed to run %s: %w", common.GetReflectTypeName(projectsZonesClustersDeleteCall), err)
	}
	log.Printf("Finished operation %s", operation.Name)
	return nil
}
