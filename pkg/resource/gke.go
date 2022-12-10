package resource

import (
	"context"
	"fmt"
	"google.golang.org/api/container/v1"
	"log"
)

type GkeCluster struct {
	ClusterName string `json:"clusterName" yaml:"clusterName"`
}

func CreateGkeCluster(gkeCluster GkeCluster) error {
	ctx := context.Background()
	containerService, err := container.NewService(ctx)
	if err != nil {
		return fmt.Errorf("failed to new container service client: %w", err)
	}
	createClusterRequest := &container.CreateClusterRequest{
		Parent: "projects/myproject001-367500/locations/us-central1-c",
		Cluster: &container.Cluster{
			Name:             "cluster-2",
			InitialNodeCount: 1,
		},
	}
	projectsZonesClustersCreateCall := containerService.Projects.Zones.Clusters.Create("", "", createClusterRequest)
	operation, err := projectsZonesClustersCreateCall.Do()
	if err != nil {
		return fmt.Errorf("failed to run projectsZonesClustersCreateCall.Do(): %w", err)
	}
	log.Printf("Finished operation %s", operation.Name)
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
		return fmt.Errorf("failed to run projectsZonesClustersDeleteCall.Do(): %w", err)
	}
	log.Printf("Finished operation %s", operation.Name)
	return nil
}
