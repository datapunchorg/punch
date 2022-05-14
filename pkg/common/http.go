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

package common

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func WaitHttpUrlReady(url string, maxWait time.Duration, retryInterval time.Duration) error {
	finalErr := RetryUntilTrue(func() (bool, error) {
		err := checkHttpUrl(url)
		if err != nil {
			log.Printf("Http url %s not ready: %s", url, err.Error())
			return false, nil
		}
		return true, nil
	}, maxWait, retryInterval)
	return finalErr
}

func PostHttpAsJsonParseResponse(url string, header http.Header, skipVerifyTlsCertificate bool, request interface{}, response interface{}) error {
	responseBytes, err := PostHttpAsJson(url, header, skipVerifyTlsCertificate, request)
	if err != nil {
		return err
	}
	err = json.Unmarshal(responseBytes, response)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response from url %s: %s, response body: %s", url, err.Error(), string(responseBytes))
	}
	return nil
}

func PostHttpAsJson(url string, header http.Header, skipVerifyTlsCertificate bool, request interface{}) ([]byte, error) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request as json: %s", err.Error())
	}
	return PostHttpWithHeader(url, header, skipVerifyTlsCertificate, requestBytes)
}

func PostHttpWithHeader(url string, header http.Header, skipVerifyTlsCertificate bool, requestBytes []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(requestBytes))
	if header != nil {
		req.Header = header
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create post request for url %s: %s", url, err.Error())
	}
	return sendHttpRequest(url, skipVerifyTlsCertificate, req)
}

func GetHttpAsJsonParseResponse(url string, header http.Header, skipVerifyTlsCertificate bool, response interface{}) error {
	responseBytes, err := GetHttp(url, header, skipVerifyTlsCertificate)
	if err != nil {
		return err
	}
	err = json.Unmarshal(responseBytes, response)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response from url %s: %s, response body: %s", url, err.Error(), string(responseBytes))
	}
	return nil
}

func GetHttp(url string, header http.Header, skipVerifyTlsCertificate bool) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, bytes.NewReader([]byte{}))
	if err != nil {
		return nil, fmt.Errorf("failed to create get request for url %s: %s", url, err.Error())
	}
	if header != nil {
		req.Header = header
	}
	return sendHttpRequest(url, skipVerifyTlsCertificate, req)
}

func sendHttpRequest(url string, skipVerifyTlsCertificate bool, req *http.Request) ([]byte, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerifyTlsCertificate},
	}
	client := &http.Client{Transport: transport}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to url %s: %s", url, err.Error())
	}
	defer response.Body.Close()
	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response data from url %s: %s", url, err.Error())
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= 300 {
		return nil, fmt.Errorf("got bad response status %d from url %s, response body: %s", response.StatusCode, url, string(responseBytes))
	}
	return responseBytes, nil
}

func checkHttpUrl(url string) error {
	req, err := http.NewRequest(http.MethodGet, url, bytes.NewReader([]byte{}))
	if err != nil {
		return fmt.Errorf("failed to create get request for url %s: %s", url, err.Error())
	}
	req.Close = true
	_, err = sendHttpRequest(url, true, req)
	return err
}
