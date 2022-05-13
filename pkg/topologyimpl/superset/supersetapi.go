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
	"fmt"
	"github.com/datapunchorg/punch/pkg/common"
	"net/http"
)

type loginRequest struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Provider string `json:"provider" yaml:"provider"`
}

type loginResponse struct {
	AccessToken string `json:"access_token" yaml:"access_token"`
}


func GetAccessToken(supersetUrl string, user string, password string) error {
	loginUrl := fmt.Sprintf("%s/api/v1/security/login", supersetUrl)
	request := loginRequest{
		Username: user,
		Password: password,
		Provider: "db",
	}
	header := http.Header{
		"Content-Type": []string{"application/json"},
	}
	response := loginResponse{}
	err := common.PostHttpAsJsonParseResponse(loginUrl, header, true, request, &response)
	if err != nil {
		return fmt.Errorf("failed to get access token from %s: %s", loginUrl, err.Error())
	}
	return nil
}
