package resource

import (
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"github.com/datapunchorg/punch/pkg/framework"
	"github.com/datapunchorg/punch/pkg/gcplib"

	"context"
	"fmt"
	"github.com/datapunchorg/punch/pkg/common"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/googleapi"
	"log"
	"strings"
	"time"
)

type GkeCluster struct {
	ClusterName      string `json:"clusterName" yaml:"clusterName"`
	InitialNodeCount int64  `json:"initialNodeCount" yaml:"initialNodeCount"`
	MachineType      string `json:"machineType" yaml:"machineType"`
}

func GetGcpFirstProjectId() (string, error) {
	ctx := context.Background()

	foldersClient, err := resourcemanager.NewFoldersClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create folders client: %w", err)
	}
	listFoldersRequest := resourcemanagerpb.ListFoldersRequest{}
	folderIterator := foldersClient.ListFolders(ctx, &listFoldersRequest)
	folder, err := folderIterator.Next()
	if err != nil {
		return "", fmt.Errorf("failed to call projectIterator.Next(): %w", err)
	}
	log.Printf("%v", folder)

	c, err := resourcemanager.NewProjectsClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create projects client: %w", err)
	}
	defer c.Close()

	listProjectsRequest := resourcemanagerpb.ListProjectsRequest{
		Parent: "folders/",
	}
	projectIterator := c.ListProjects(ctx, &listProjectsRequest)
	project, err := projectIterator.Next()
	if err != nil {
		return "", fmt.Errorf("failed to call projectIterator.Next(): %w", err)
	}
	return project.ProjectId, nil
}

func CreateGkeCluster(projectId string, zone string, gkeCluster GkeCluster) error {
	clusterName := gkeCluster.ClusterName

	ctx := context.Background()
	containerService, err := container.NewService(ctx)
	if err != nil {
		return fmt.Errorf("failed to new container service client: %w", err)
	}
	createClusterRequest := &container.CreateClusterRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, zone),
		Cluster: &container.Cluster{
			Name:             clusterName,
			InitialNodeCount: gkeCluster.InitialNodeCount,
			NodeConfig: &container.NodeConfig{
				MachineType: gkeCluster.MachineType,
			},
			// Autoscaling: &container.ClusterAutoscaling{},
		},
	}
	projectsZonesClustersCreateCall := containerService.Projects.Zones.Clusters.Create("", "", createClusterRequest)
	_, err = projectsZonesClustersCreateCall.Do()
	if err != nil {
		if !IsErrorGoogleApiAlreadyExits(err) {
			return fmt.Errorf("failed to run %s: %w", common.GetReflectTypeName(projectsZonesClustersCreateCall), err)
		} else {
			log.Printf("Ignore cluster already exist error for %s: %s", clusterName, err.Error())
		}
	}
	log.Printf("Finished operation %s", common.GetReflectTypeName(projectsZonesClustersCreateCall))

	waitClusterReadyErr := common.RetryUntilTrue(func() (bool, error) {
		targetClusterStatus := "RUNNING"
		projectsZonesClustersGetCall := containerService.Projects.Zones.Clusters.Get(projectId, zone, clusterName)
		getClusterResult, err := projectsZonesClustersGetCall.Do()
		if err != nil {
			return false, fmt.Errorf("failed to run %s: %w", common.GetReflectTypeName(projectsZonesClustersGetCall), err)
		}
		ready := strings.EqualFold(getClusterResult.Status, targetClusterStatus)
		if !ready {
			log.Printf("GKE cluster %s not ready (in status: %s), waiting... (GKE may take 5+ minutes to be ready, thanks for your patience)", clusterName, getClusterResult.Status)
		} else {
			log.Printf("GKE cluster %s is ready (in status: %s)", clusterName, getClusterResult.Status)
		}
		return ready, nil
	}, 30*time.Minute, 30*time.Second)

	if waitClusterReadyErr != nil {
		return fmt.Errorf("failed to wait ready for cluster %s: %s", clusterName, waitClusterReadyErr.Error())
	}

	return nil
}

func DeleteGkeCluster(projectId string, zone string, clusterId string) error {
	ctx := context.Background()
	containerService, err := container.NewService(ctx)
	if err != nil {
		return fmt.Errorf("failed to new container service client: %w", err)
	}
	projectsZonesClustersDeleteCall := containerService.Projects.Zones.Clusters.Delete(projectId, zone, clusterId)
	_, err = projectsZonesClustersDeleteCall.Do()
	if err != nil {
		return fmt.Errorf("failed to run %s: %w", common.GetReflectTypeName(projectsZonesClustersDeleteCall), err)
	}
	log.Printf("Finished operation %s", common.GetReflectTypeName(projectsZonesClustersDeleteCall))

	// TODO wait cluster deleted

	return nil
}

func IsErrorGoogleApiAlreadyExits(err error) bool {
	if err == nil {
		return false
	}
	googleapiErr, ok := err.(*googleapi.Error)
	if !ok {
		return false
	}
	if googleapiErr.Code == 409 &&
		len(googleapiErr.Errors) > 0 &&
		strings.EqualFold(googleapiErr.Errors[0].Reason, "alreadyExists") {
		return true
	}
	return false
}

func GetGkeNginxLoadBalancerUrls(commandEnvironment framework.CommandEnvironment, projectId, zone, clusterName string, nginxNamespace string, nginxServiceName string, explicitHttpsPort int32) ([]string, error) {
	hostPorts, err := gcplib.GetServiceLoadBalancerHostPorts(commandEnvironment.Get(framework.CmdEnvKubeConfig), projectId, zone, clusterName, nginxNamespace, nginxServiceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get load balancer urls for nginx controller service %s in namespace %s: %s", nginxServiceName, nginxNamespace, err.Error())
	}

	dnsNamesMap := make(map[string]bool, len(hostPorts))
	for _, entry := range hostPorts {
		dnsNamesMap[entry.Host] = true
	}
	dnsNames := make([]string, 0, len(dnsNamesMap))
	for k := range dnsNamesMap {
		dnsNames = append(dnsNames, k)
	}

	//err = awslib.WaitLoadBalancersReadyByDnsNames(region, dnsNames)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to wait and get load balancer urls: %s", err.Error())
	//}

	urls := make([]string, 0, len(hostPorts))
	for _, entry := range hostPorts {
		if entry.Port == 443 {
			urls = append(urls, fmt.Sprintf("https://%s", entry.Host))
		} else if entry.Port == explicitHttpsPort {
			urls = append(urls, fmt.Sprintf("https://%s:%d", entry.Host, entry.Port))
		} else if entry.Port == 80 {
			urls = append(urls, fmt.Sprintf("http://%s", entry.Host))
		} else {
			urls = append(urls, fmt.Sprintf("http://%s:%d", entry.Host, entry.Port))
		}
	}

	return urls, nil
}
