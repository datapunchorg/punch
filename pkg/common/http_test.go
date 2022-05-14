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
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestWaitHttpUrlReady(t *testing.T) {
	err := checkHttpUrl("http://not_exist_url_abc_123")
	assert.NotNil(t, err)
	err = checkHttpUrl("https://www.google.com")
	assert.Nil(t, err)

	err = WaitHttpUrlReady("http://not_exist_url_abc_123", 10 * time.Millisecond, 1 * time.Millisecond)
	assert.NotNil(t, err)
	err = WaitHttpUrlReady("https://www.google.com", 10 * time.Millisecond, 1 * time.Millisecond)
	assert.Nil(t, err)
}

func TestGetHttp(t *testing.T) {
	bytes, err := GetHttp("http://not_exist_url_abc_123", nil, true)
	assert.NotNil(t, err)
	assert.Nil(t, bytes)
	bytes, err = GetHttp("https://www.google.com", nil, true)
	assert.Nil(t, err)
	assert.NotNil(t, bytes)
}