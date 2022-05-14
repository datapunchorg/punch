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

package superset

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// var supersetUrl = ""
var supersetUrl = "http://aaf74168a724448549a2ac4386f97d81-1339844253.us-west-1.elb.amazonaws.com:8088"

func TestGetAccessToken(t *testing.T) {
	if supersetUrl == "" {
		return
	}
	token, err := GetAccessToken(supersetUrl, "admin", "admin")
	assert.Nil(t, err)
	assert.NotEqual(t, "", token)
}

func TestAddDatabase(t *testing.T) {
	if supersetUrl == "" {
		return
	}
	accessToken, err := GetAccessToken(supersetUrl, "admin", "admin")
	assert.Nil(t, err)
	assert.NotEqual(t, "", accessToken)

	csrfToken, err := GetCsrfToken(supersetUrl, accessToken)
	assert.Nil(t, err)
	assert.NotEqual(t, "", csrfToken)

	databaseInfo := DatabaseInfo{
		DatabaseName: "punch-unit-test-01",
		Engine: "hive",
		SqlalchemyUri: "hive://hive@kyuubi-svc.kyuubi-01.svc.cluster.local:10009/default",
	}
	err = AddDatabase(supersetUrl, csrfToken, databaseInfo)
	assert.Nil(t, err)
}