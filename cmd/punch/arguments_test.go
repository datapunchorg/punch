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

package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_getKind(t *testing.T) {
	str := `
kind: Kind1
metadata:
    name: name1
`
	assert.Equal(t, "Kind1", getKind([]byte(str)))

	assert.Equal(t, "", getKind([]byte("")))
	assert.Equal(t, "", getKind([]byte("abc")))
	assert.Equal(t, "", getKind([]byte("kind:")))
	assert.Equal(t, "", getKind([]byte("Kind: Kind1")))
}
