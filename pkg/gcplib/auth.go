package gcplib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/datapunchorg/punch/pkg/common"
	"golang.org/x/oauth2/google"
	oauthsvc "google.golang.org/api/oauth2/v2"
	"google.golang.org/grpc/credentials/oauth"
	"io"
	"net/http"
	"os"
	"path"
	"time"
)

type ApplicationDefaultCredentials struct {
	ClientId     string `json:"client_id" yaml:"client_id"`
	ClientSecret string `json:"client_secret" yaml:"client_secret"`
	RefreshToken string `json:"refresh_token" yaml:"refresh_token"`
}

type GetOauthTokenRequest struct {
	ClientId     string `json:"client_id" yaml:"client_id"`
	ClientSecret string `json:"client_secret" yaml:"client_secret"`
	RefreshToken string `json:"refresh_token" yaml:"refresh_token"`
	GrantType    string `json:"grant_type" yaml:"grant_type"`
}

type GetOauthTokenResponse struct {
	AccessToken string `json:"access_token" yaml:"access_token"`
}

func GetCurrentUserAccessToken() (string, error) {
	// return getCurrentUserAccessTokenUsingADCJsonFile()
	return getCurrentUserAccessTokenUsingDefaultTokenSource()
}

// also see https://gist.github.com/salrashid123/9810b45387adbd45a22bb4e17bdbe9d5

func getCurrentUserAccessTokenUsingADCJsonFile() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home dir: %w", err)
	}
	credentialFileRelativePath := ".config/gcloud/application_default_credentials.json"
	filePath := path.Join(homeDir, credentialFileRelativePath)
	exists, err := FileExists(filePath)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("user credential file not exists: %s", filePath)
	}
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	var cred ApplicationDefaultCredentials
	err = json.Unmarshal(fileBytes, &cred)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal content in file %s: %w", filePath, err)
	}
	if cred.ClientId == "" {
		return "", fmt.Errorf("client-id is empty in file %s: %w", filePath, err)
	}
	if cred.ClientSecret == "" {
		return "", fmt.Errorf("client-secret is empty in file %s: %w", filePath, err)
	}
	if cred.RefreshToken == "" {
		return "", fmt.Errorf("refresh-token is empty in file %s: %w", filePath, err)
	}
	url := "https://oauth2.googleapis.com/token"
	request := GetOauthTokenRequest{
		ClientId:     cred.ClientId,
		ClientSecret: cred.ClientSecret,
		RefreshToken: cred.RefreshToken,
		GrantType:    "refresh_token",
	}
	response, err := PostHttpJsonRequestJsonResponse[GetOauthTokenRequest, GetOauthTokenResponse](url, request, 60*time.Second, true)
	if err != nil {
		return "", err
	}
	if response.AccessToken == "" {
		return "", fmt.Errorf("got empty access token from %s", url)
	}
	return response.AccessToken, nil
}

func getCurrentUserAccessTokenUsingDefaultTokenSource() (string, error) {
	defaultTokenSource, err := google.DefaultTokenSource(context.Background(), oauthsvc.UserinfoEmailScope)
	if err != nil {
		return "", fmt.Errorf("failed to get default token source: %w", err)
	}
	oauthTokenSource := oauth.TokenSource{
		TokenSource: defaultTokenSource,
	}
	token, err := oauthTokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get token from oauth token source: %w", err)
	}
	return token.AccessToken, nil
}

func FileExists(filePath string) (bool, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, fmt.Errorf("failed to check file exists for %s: %w", filePath, err)
		}
	}
	return !info.IsDir(), nil
}

func PostHttpJsonRequestJsonResponse[TRequest any, TResponse any](url string, request TRequest, timeout time.Duration, readResponseBodyOnError bool) (TResponse, error) {
	var response TResponse

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return response, fmt.Errorf("failed to serialize request to json: %w", err)
	}

	requestBytesReader := bytes.NewReader(requestBytes)
	var requestBody io.Reader = requestBytesReader
	req, err := http.NewRequest(http.MethodPost, url, requestBody)
	if err != nil {
		return response, fmt.Errorf("failed to create post request for %s: %w", url, err)
	}

	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{
		Timeout: timeout,
	}
	httpResponse, err := httpClient.Do(req)
	if err != nil {
		return response, fmt.Errorf("failed to post %s: %w", url, err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return response, ErrorBadHttpStatus(url, httpResponse, readResponseBodyOnError)
	}

	responseBytes, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return response, fmt.Errorf("failed to read response body for %s: %w", url, err)
	}

	err = json.Unmarshal(responseBytes, &response)
	if err != nil {
		return response, fmt.Errorf("failed to parse response %s from %s: %w, response: %s", common.GetReflectTypeName(response), url, err, string(responseBytes))
	}

	return response, nil
}

func ErrorBadHttpStatus(url string, response *http.Response, readBody bool) error {
	bodyStr := "(skip)"
	if readBody {
		bytes, err := io.ReadAll(response.Body)
		if err != nil {
			bodyStr = fmt.Sprintf("failed to read response body: %s", err.Error())
		} else {
			bodyStr = string(bytes)
		}
	}
	return fmt.Errorf("got bad response status %s from %s, response body: %s", response.Status, url, bodyStr)
}
