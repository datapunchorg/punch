package resource

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// Enable Cloud Resource Manager API: https://console.developers.google.com/apis/api/cloudresourcemanager.googleapis.com/overview?project=733425931108

// gcloud projects \
//    add-iam-policy-binding \
//    myproject001-367500 --member \
//    serviceAccount:logging-quickstart@myproject001-367500.iam.gserviceaccount.com \
//    --role roles/container.admin
//
// gcloud projects \
//    add-iam-policy-binding \
//    myproject001-367500 --member \
//    serviceAccount:logging-quickstart@myproject001-367500.iam.gserviceaccount.com \
//    --role roles/container.clusterAdmin
//
// gcloud projects \
//    add-iam-policy-binding \
//    myproject001-367500 --member \
//    serviceAccount:logging-quickstart@myproject001-367500.iam.gserviceaccount.com \
//    --role roles/iam.serviceAccountUser

// export GOOGLE_APPLICATION_CREDENTIALS=/Users/admin/temp/logging-key.json

func TestCreateGkeCluster(t *testing.T) {
	projectId, err := GetGcpFirstProjectId()
	require.Nil(t, err)
	zone := "us-central1-c"
	gkeCluster := GkeCluster{
		ClusterName: "cluster-1",
	}
	err = CreateGkeCluster(projectId, zone, gkeCluster)
	require.Nil(t, err)
}

func TestDeleteGkeCluster(t *testing.T) {
	err := DeleteGkeCluster("myproject001-367500", "us-central1-c", "cluster-1")
	require.Nil(t, err)
}
