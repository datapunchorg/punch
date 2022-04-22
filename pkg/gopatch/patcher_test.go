package gopatch

import(
  "reflect"
  "strings"
  "testing"
)

func TestPatcher(t *testing.T) {

  type TestDouble struct {
    Field1  string
    Field2  int
  }

  type TestEmbedded struct {
    Field1  string
    Field2  int
    Field3  bool
    Field4  TestDouble
  }

  type TestStruct struct {
    Field1  string        `bson:"field1"  json:"field_1"`
    Field2  int
    Field3  bool          `gopatch:"-"`
    Field4  TestEmbedded
    Field5  TestEmbedded  `gopatch:"patch"`
    Field6  TestEmbedded  `gopatch:"replace"`
  }

  t.Run("embed-path", func(t *testing.T) {

    cfg := PatcherConfig{
      EmbedPath: "test",
    }

    patcher := New(cfg)

    testInstance := TestStruct{}

    result, err := patcher.Patch(&testInstance, map[string]interface{}{
      "Field2": 255,
    })

    // Test for unexpected errors.
    if err != nil {
      t.Errorf("Unexpected patch error: %q", err.Error())
      return
    }

    // Test to see if the instance was patched.
    if testInstance.Field1 != "" || testInstance.Field2 != 255 || testInstance.Field3 {
      t.Errorf("Expected patch to only patch Field2. Patch affected struct so: %v", testInstance)
      return
    }

    // Test to see if the resulting update map is correct.
    if v, e := result.Map["test.Field2"]; !e || v != 255 {
      t.Errorf("Expected patch result map to contain exactly \"test.Field2\": 255. Contained %v", result.Map)
      return
    }
  })

  t.Run("patch-source", func(t *testing.T) {

    cfg := PatcherConfig{
      PatchSource: "json",
    }

    patcher := New(cfg)

    testInstance := TestStruct{}

    result, err := patcher.Patch(&testInstance, map[string]interface{}{
      "field_1": "test",
    })

    // Test for unexpected errors.
    if err != nil {
      t.Errorf("Unexpected patch error: %q", err.Error())
      return
    }

    // Test to see if the instance was patched.
    if testInstance.Field1 != "test" || testInstance.Field2 != 0 || testInstance.Field3 {
      t.Errorf("Expected patch to only patch Field1. Patch affected struct so: %v", testInstance)
      return
    }

    // Test to see if the instance field list is correct.
    if len(result.Fields) != 1 || result.Fields[0] != "Field1" {
      t.Errorf("Expected patch result fields to contain exactly \"Field1\". Contained [%v]", strings.Join(result.Fields, ", "))
      return
    }

    // Test to see if the resulting update map is correct.
    if v, e := result.Map["Field1"]; !e || v != "test" {
      t.Errorf("Expected patch result map to contain exactly \"Field1\": \"test\". Contained %v", result.Map)
      return
    }
  })

  t.Run("patch-errors", func(t *testing.T) {

    cfg := PatcherConfig{
      PatchSource: "msgpack",
      PatchErrors: true,
    }

    patcher := New(cfg)

    testInstance := TestStruct{}

    _, err := patcher.Patch(&testInstance, map[string]interface{}{
      "field_1": "test",
    })

    // Test for unexpected errors.
    if err == nil {
      t.Errorf("Expected patch error, but didn't get one.")
      return
    }
  })

  t.Run("map-source", func(t *testing.T) {

    cfg := PatcherConfig{
      UpdatedMapSource: "bson",
    }

    patcher := New(cfg)

    testInstance := TestStruct{}

    result, err := patcher.Patch(&testInstance, map[string]interface{}{
      "Field1": "test",
    })

    // Test for unexpected errors.
    if err != nil {
      t.Errorf("Unexpected patch error: %q", err.Error())
      return
    }

    // Test to see if the instance field list is correct.
    if len(result.Fields) != 1 || result.Fields[0] != "Field1" {
      t.Errorf("Expected patch result fields to contain exactly \"Field1\". Contained [%v]", strings.Join(result.Fields, ", "))
      return
    }

    // Test to see if the resulting update map is correct.
    if v, e := result.Map["field1"]; !e || v != "test" {
      t.Errorf("Expected patch result map to contain exactly \"field1\": \"test\". Contained %v", result.Map)
      return
    }
  })

  t.Run("map-errors", func(t *testing.T) {

    cfg := PatcherConfig{
      UpdatedMapSource: "msgpack",
      UpdatedMapErrors: true,
    }

    patcher := New(cfg)

    testInstance := TestStruct{}

    _, err := patcher.Patch(&testInstance, map[string]interface{}{
      "Field1": "test",
    })

    // Test for unexpected errors.
    if err == nil {
      t.Errorf("Expected patch error, but didn't get one.")
      return
    }
  })

  t.Run("field-source", func(t *testing.T) {

    cfg := PatcherConfig{
      UpdatedFieldSource: "bson",
      UpdatedFieldErrors: true,
    }

    patcher := New(cfg)

    testInstance := TestStruct{}

    result, err := patcher.Patch(&testInstance, map[string]interface{}{
      "Field1": "test",
    })

    // Test for unexpected errors.
    if err != nil {
      t.Errorf("Unexpected patch error: %q", err.Error())
      return
    }

    // Test to see if the instance field list is correct.
    if len(result.Fields) != 1 || result.Fields[0] != "field1" {
      t.Errorf("Expected patch result fields to contain exactly \"field1\". Contained [%v]", strings.Join(result.Fields, ", "))
      return
    }

    // Test to see if the resulting update map is correct.
    if v, e := result.Map["Field1"]; !e || v != "test" {
      t.Errorf("Expected patch result map to contain exactly \"Field1\": \"test\". Contained %v", result.Map)
      return
    }
  })

  t.Run("field-errors", func(t *testing.T) {

    cfg := PatcherConfig{
      UpdatedFieldSource: "msgpack",
      UpdatedFieldErrors: true,
    }

    patcher := New(cfg)

    testInstance := TestStruct{}

    _, err := patcher.Patch(&testInstance, map[string]interface{}{
      "Field1": "test",
    })

    // Test for unexpected errors.
    if err == nil {
      t.Errorf("Expected patch error, but didn't get one.")
      return
    }
  })

  t.Run("permitted-only", func(t *testing.T) {

    cfg := PatcherConfig{
      PermittedFields: []string{"Field1"},
    }

    patcher := New(cfg)

    testInstance := TestStruct{}

    result, err := patcher.Patch(&testInstance, map[string]interface{}{
      "Field1": "test",
      "Field2": 255,
    })

    // Test for unexpected errors.
    if err != nil {
      t.Errorf("Unexpected patch error: %q", err.Error())
      return
    }

    // Test to see if the instance was patched.
    if testInstance.Field1 != "test" || testInstance.Field2 != 0 || testInstance.Field3 {
      t.Errorf("Expected patch to only patch Field1. Patch affected struct so: %v", testInstance)
      return
    }

    // Test to see if the instance field list is correct, and only contains the permitted field.
    if len(result.Unpermitted) != 1 || result.Unpermitted[0] != "Field2" {
      t.Errorf("Expected patch result unpermitted to contain exactly \"Field2\". Contained [%v]", strings.Join(result.Unpermitted, ", "))
      return
    }

    // Test to see if the resulting update map is correct, and only contains the permitted field.
    if v, e := result.Map["Field1"]; !e || v != "test" || len(result.Map) > 1 {
      t.Errorf("Expected patch result map to contain exactly \"Field1\": \"test\". Contained %v", result.Map)
      return
    }
  })

  t.Run("permitted-error", func(t *testing.T) {

    cfg := PatcherConfig{
      PermittedFields: []string{"Field1"},
      UnpermittedErrors: true,
    }

    patcher := New(cfg)

    testInstance := TestStruct{}

    _, err := patcher.Patch(&testInstance, map[string]interface{}{
      "Field1": "test",
      "Field2": 255,
    })

    // Test for unexpected errors.
    if err == nil {
      t.Errorf("Expected patch error, but didn't get one.")
      return
    }
  })

  t.Run("unpermitted-by-tag", func(t *testing.T) {

    cfg := PatcherConfig{}

    patcher := New(cfg)

    testInstance := TestStruct{}

    result, err := patcher.Patch(&testInstance, map[string]interface{}{
      "Field1": "test",
      "Field3": true,
    })

    // Test for unexpected errors.
    if err != nil {
      t.Errorf("Unexpected patch error: %q", err.Error())
      return
    }

    // Test to see if the instance was patched.
    if testInstance.Field1 != "test" || testInstance.Field2 != 0 || testInstance.Field3 {
      t.Errorf("Expected patch to only patch Field1. Patch affected struct so: %v", testInstance)
      return
    }

    // Test to see if the instance field list is correct, and only contains the permitted field.
    if len(result.Unpermitted) != 1 || result.Unpermitted[0] != "Field3" {
      t.Errorf("Expected patch result unpermitted to contain exactly \"Field3\". Contained [%v]", strings.Join(result.Unpermitted, ", "))
      return
    }

    // Test to see if the resulting update map is correct, and only contains the permitted field.
    if v, e := result.Map["Field1"]; !e || v != "test" || len(result.Map) > 1 {
      t.Errorf("Expected patch result map to contain exactly \"Field1\": \"test\". Contained %v", result.Map)
      return
    }
  })

  t.Run("embedded-without-tag", func(t *testing.T) {

    cfg := PatcherConfig{}

    patcher := New(cfg)

    testInstance := TestStruct{
      Field4: TestEmbedded{
        Field1: "test",
        Field2: 1,
      },
    }

    result, err := patcher.Patch(&testInstance, map[string]interface{}{
      "Field4": map[string]interface{}{
        "Field2": 255,
      },
    })

    // Test for unexpected errors.
    if err != nil {
      t.Errorf("Unexpected patch error: %q", err.Error())
      return
    }

    // Test to see if the instance was patched.
    if testInstance.Field4.Field1 != "test" || testInstance.Field4.Field2 != 255 {
      t.Errorf("Expected patch to only patch Field4.Field2. Patch affected struct so: %v", testInstance)
      return
    }

    // Test to see if the resulting update map is correct, and only contains the permitted field.
    if v, e := result.Map["Field4.Field2"]; !e || v != 255 || len(result.Map) > 1 {
      t.Errorf("Expected patch result map to contain exactly \"Field4.Field2\": \"test\". Contained %v", result.Map)
      return
    }
  })

  t.Run("embedded-tagged-patch", func(t *testing.T) {

    cfg := PatcherConfig{}

    patcher := New(cfg)

    testInstance := TestStruct{
      Field5: TestEmbedded{
        Field1: "test",
        Field2: 1,
      },
    }

    result, err := patcher.Patch(&testInstance, map[string]interface{}{
      "Field5": map[string]interface{}{
        "Field2": 255,
      },
    })

    // Test for unexpected errors.
    if err != nil {
      t.Errorf("Unexpected patch error: %q", err.Error())
      return
    }

    // Test to see if the instance was patched.
    if testInstance.Field5.Field1 != "test" || testInstance.Field5.Field2 != 255 {
      t.Errorf("Expected patch to only patch Field5.Field2. Patch affected struct so: %v", testInstance)
      return
    }

    // Test to see if the resulting update map is correct, and only contains the permitted field.
    if v, e := result.Map["Field5.Field2"]; !e || v != 255 || len(result.Map) > 1 {
      t.Errorf("Expected patch result map to contain exactly \"Field5.Field2\": 255. Contained %v", result.Map)
      return
    }
  })

  t.Run("embedded-tagged-replace", func(t *testing.T) {

    cfg := PatcherConfig{}

    patcher := New(cfg)

    testInstance := TestStruct{
      Field5: TestEmbedded{
        Field1: "test",
        Field2: 1,
        Field3: true,
      },
    }

    result, err := patcher.Patch(&testInstance, map[string]interface{}{
      "Field6": map[string]interface{}{
        "Field2": 255,
      },
    })

    // Test for unexpected errors.
    if err != nil {
      t.Errorf("Unexpected patch error: %q", err.Error())
      return
    }

    // Test to see if the instance was patched.
    if testInstance.Field6.Field1 != "" || testInstance.Field6.Field2 != 255 || testInstance.Field6.Field3 {
      t.Errorf("Expected patch to only patch Field6.Field2. Patch affected struct so: %v", testInstance)
      return
    }

    // Test to see if the resulting update map is correct, and only contains the permitted field.
    if v, e := result.Map["Field6"]; !e || reflect.ValueOf(v).Kind() != reflect.Map || len(result.Map) > 1 {
      t.Errorf("Expected patch result map to contain exactly \"Field6\": map. Contained %v", result.Map)
      return
    }
  })

  t.Run("embedded-double", func(t *testing.T) {

    cfg := PatcherConfig{}

    patcher := New(cfg)

    testInstance := TestStruct{
      Field4: TestEmbedded{
        Field4: TestDouble{
          Field1: "test",
        },
      },
    }

    result, err := patcher.Patch(&testInstance, map[string]interface{}{
      "Field4": map[string]interface{}{
        "Field4": map[string]interface{}{
          "Field2": 255,
        },
      },
    })

    // Test for unexpected errors.
    if err != nil {
      t.Errorf("Unexpected patch error: %q", err.Error())
      return
    }

    // Test to see if the instance was patched.
    if testInstance.Field4.Field4.Field1 != "test" || testInstance.Field4.Field4.Field2 != 255 {
      t.Errorf("Expected patch to only patch Field4.Field2. Patch affected struct so: %v", testInstance)
      return
    }

    // Test to see if the resulting update map is correct, and only contains the permitted field.
    if v, e := result.Map["Field4.Field4.Field2"]; !e || v != 255 || len(result.Map) > 1 {
      t.Errorf("Expected patch result map to contain exactly \"Field4.Field4.Field2\": 255. Contained %v", result.Map)
      return
    }
  })
}