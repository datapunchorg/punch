package resource

import (
	"github.com/stretchr/testify/require"
	"testing"
)

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
	projectId := "myproject001-367500"
	zone := "us-central1-c"
	gkeCluster := GkeCluster{
		ClusterName: "cluster-1",
	}
	err := CreateGkeCluster(projectId, zone, gkeCluster)
	require.Nil(t, err)
}

func TestDeleteGkeCluster(t *testing.T) {
	err := DeleteGkeCluster("myproject001-367500", "us-central1-c", "cluster-1")
	require.Nil(t, err)
}
