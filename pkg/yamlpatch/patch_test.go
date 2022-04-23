package yamlpatch

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPatchStringValue(t *testing.T) {
	var value interface{} = "new"
	ops := []Operation{
		{
			Op: OpReplace,
			Path: "/f01/f02",
			Value: NewNode(&value),
		},
	}
	patch := Patch(ops)
	oldYaml := `foo: 123
f01:
  f02: old
`
	patched, err := patch.Apply([]byte(oldYaml))
	assert.Nil(t, err)
	newYaml := `f01:
    f02: new
foo: 123
`
	assert.Equal(t, newYaml, string(patched))
}
