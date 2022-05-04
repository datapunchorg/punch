/*
Copyright 2022 DataPunch Project

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
	"fmt"
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

func checkHttpUrl(url string) error {
	req, err := http.NewRequest(http.MethodGet, url, bytes.NewReader([]byte{}))
	if err != nil {
		return fmt.Errorf("failed to create get http request for %s: %s", url, err.Error())
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get http %s: %s", url, err.Error())
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("got bad response from http %s: %d", url, response.StatusCode)
	}
	return nil
}
