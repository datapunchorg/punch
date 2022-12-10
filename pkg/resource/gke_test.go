package resource

import "testing"

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
	gkeCluster := GkeCluster{
		ClusterName: "cluster-1",
	}
	CreateGkeCluster(gkeCluster)
}

func TestDeleteGkeCluster(t *testing.T) {
	DeleteGkeCluster("myproject001-367500", "us-central1-c", "cluster-1")
}
