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

type csrfResponse struct {
	Result string `json:"result" yaml:"result"`
}

type databaseResponse struct {
	Id int64 `json:"id" yaml:"id"`
}

func GetAccessToken(supersetUrl string, user string, password string) (string, error) {
	requestUrl := fmt.Sprintf("%s/api/v1/security/login", supersetUrl)
	request := loginRequest{
		Username: user,
		Password: password,
		Provider: "db",
	}
	header := http.Header{
		"Content-Type": []string{"application/json"},
	}
	response := loginResponse{}
	err := common.PostHttpAsJsonParseResponse(requestUrl, header, true, request, &response)
	if err != nil {
		return "", fmt.Errorf("failed to get access token from %s: %s", requestUrl, err.Error())
	}
	return response.AccessToken, nil
}

func GetCsrfToken(supersetUrl string, accessToken string) (string, error) {
	requestUrl := fmt.Sprintf("%s/api/v1/security/csrf_token/", supersetUrl)
	header := http.Header{
		"Authorization": []string{
			fmt.Sprintf("Bearer %s", accessToken),
		},
	}
	response := csrfResponse{}
	err := common.GetHttpAsJsonParseResponse(requestUrl, header, true, &response)
	if err != nil {
		return "", fmt.Errorf("failed to get CSRF token from %s: %s", requestUrl, err.Error())
	}
	return response.Result, nil
}

func AddDatabase(supersetUrl string, accessToken string, databaseInfo DatabaseInfo) error {
	//loginUrl := fmt.Sprintf("%s/api/v1/security/login", supersetUrl)
	requestUrl := fmt.Sprintf("%s/api/v1/database/", supersetUrl)
	request := databaseInfo
	header := http.Header{
		"Authorization": []string{
			fmt.Sprintf("Bearer %s", accessToken),
		},
		"Content-Type": []string{"application/json"},
	}
	response := databaseResponse{}
	err := common.PostHttpAsJsonParseResponse(requestUrl, header, true, request, &response)
	if err != nil {
		return fmt.Errorf("failed to add database to %s: %s", requestUrl, err.Error())
	}
	return nil
}

func AddDatabaseWithCsrfToken(supersetUrl string, csrfToken string, databaseInfo DatabaseInfo) error {
	//loginUrl := fmt.Sprintf("%s/api/v1/security/login", supersetUrl)
	requestUrl := fmt.Sprintf("%s/api/v1/database/", supersetUrl)
	request := databaseInfo
	header := http.Header{
		"X-CSRFToken": []string{csrfToken},
		"Content-Type": []string{"application/json"},
	}
	response := databaseResponse{}
	err := common.PostHttpAsJsonParseResponse(requestUrl, header, true, request, &response)
	if err != nil {
		return fmt.Errorf("failed to add database to %s: %s", requestUrl, err.Error())
	}
	return nil
}