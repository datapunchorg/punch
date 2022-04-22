package gopatch

import(
  "strings"
  "testing"
)

func TestDefault(t *testing.T) {

  type TestEmbedded struct {
    Field1  string
    Field2  int
    Field3  bool
  }

  type TestStruct struct {
    Field1  string
    Field2  int
    Field3  bool
    Field4  TestEmbedded
    Field5  *TestEmbedded
    Field6  map[string]int
  }

  t.Run("default-has-default-config", func(t *testing.T) {

    patcher := Default()

    if patcher.config.EmbedPath != defaultConfig.EmbedPath {
      t.Errorf("Expected default embed path %q but got %q", defaultConfig.EmbedPath, patcher.config.EmbedPath)
    }
    if patcher.config.PermittedFields != nil && defaultConfig.PermittedFields != nil {
      for _, g := range(patcher.config.PermittedFields) {

        found := false
        for _, d := range(defaultConfig.PermittedFields) {
          if g == d {
            found = true
            break
          }
        }
        if !found {
          t.Errorf("Expected default permitted fields [%q] but got [%q]", strings.Join(defaultConfig.PermittedFields, ", "), strings.Join(patcher.config.PermittedFields, ", "))
          break
        }
      }
      for _, d := range(defaultConfig.PermittedFields) {

        found := false
        for _, g := range(patcher.config.PermittedFields) {
          if g == d {
            found = true
            break
          }
        }
        if !found {
          t.Errorf("Expected default permitted fields [%q] but got [%q]", strings.Join(defaultConfig.PermittedFields, ", "), strings.Join(patcher.config.PermittedFields, ", "))
          break
        }
      }
    } else if (patcher.config.PermittedFields == nil || defaultConfig.PermittedFields == nil) && !(patcher.config.PermittedFields == nil && defaultConfig.PermittedFields == nil) {
      t.Errorf("Expected default permitted fields [%v] but got [%v]", defaultConfig.PermittedFields, patcher.config.PermittedFields)
    }
    if patcher.config.UnpermittedErrors != defaultConfig.UnpermittedErrors {
      t.Errorf("Expected default unpermitted-causes-errors %v but got %v", defaultConfig.UnpermittedErrors, patcher.config.UnpermittedErrors)
    }
    if patcher.config.UpdatedFieldErrors != defaultConfig.UpdatedFieldErrors {
      t.Errorf("Expected default missing-field-tag-causes-errors (fields) %v but got %v", defaultConfig.UpdatedFieldErrors, patcher.config.UpdatedFieldErrors)
    }
    if patcher.config.UpdatedFieldSource != defaultConfig.UpdatedFieldSource {
      t.Errorf("Expected default field source %q but got %q", defaultConfig.UpdatedFieldSource, patcher.config.UpdatedFieldSource)
    }
    if patcher.config.UpdatedMapErrors != defaultConfig.UpdatedMapErrors {
      t.Errorf("Expected default missing-field-tag-causes-errors (map) %v but got %v", defaultConfig.UpdatedMapErrors, patcher.config.UpdatedMapErrors)
    }
    if patcher.config.UpdatedMapSource != defaultConfig.UpdatedMapSource {
      t.Errorf("Expected default map source %q but got %q", defaultConfig.UpdatedMapSource, patcher.config.UpdatedMapSource)
    }
  })

  t.Run("default-top-level-patches", func(t *testing.T) {

    testInstance := TestStruct{}

    result, err := Default().Patch(&testInstance, map[string]interface{}{
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

    // Test to see if the instance field list is correct.
    if len(result.Fields) != 1 || result.Fields[0] != "Field2" {
      t.Errorf("Expected patch result fields to contain exactly \"Field2\". Contained [%v]", strings.Join(result.Fields, ", "))
      return
    }

    // Test to see if the resulting update map is correct.
    if v, e := result.Map["Field2"]; !e || v != 255 {
      t.Errorf("Expected patch result map to contain exactly \"Field2\": 255. Contained %v", result.Map)
      return
    }
  })

  t.Run("default-embedded-patches", func(t *testing.T) {

    testInstance := TestStruct{}

    result, err := Default().Patch(&testInstance, map[string]interface{}{
      "Field4": map[string]interface{}{
        "Field1": "test",
      },
    })

    // Test for unexpected errors.
    if err != nil {
      t.Errorf("Unexpected patch error: %q", err.Error())
      return
    }

    // Test to see if the instance was patched.
    if testInstance.Field4.Field1 != "test" || testInstance.Field4.Field2 != 0 || testInstance.Field4.Field3 {
      t.Errorf("Expected patch to only patch Field5.Field1. Patch affected struct so: %v", testInstance)
      return
    }

    // Test to see if the instance field list is correct.
    if len(result.Fields) != 1 || result.Fields[0] != "Field4.Field1" {
      t.Errorf("Expected patch result fields to contain exactly \"Field4.Field1\". Contained [%v]", strings.Join(result.Fields, ", "))
      return
    }

    // Test to see if the resulting update map is correct.
    if v, e := result.Map["Field4.Field1"]; !e || v != "test" {
      t.Errorf("Expected patch result map to contain exactly \"Field4.Field1\": \"test\". Contained %v", result.Map)
      return
    }
  })

  t.Run("default-nil-pointer-patches", func(t *testing.T) {

    testInstance := TestStruct{}

    result, err := Default().Patch(&testInstance, map[string]interface{}{
      "Field5": map[string]interface{}{
        "Field1": "test",
      },
    })

    // Test for unexpected errors.
    if err != nil {
      t.Errorf("Unexpected patch error: %q", err.Error())
      return
    }

    // Test to see if the instance was patched.
    if testInstance.Field5 == nil || testInstance.Field5.Field1 != "test" || testInstance.Field5.Field2 != 0 || testInstance.Field5.Field3 {
      t.Errorf("Expected patch to only patch Field5.Field1. Patch affected struct so: %v", testInstance)
      return
    }

    // Test to see if the resulting field list is correct.
    if len(result.Fields) != 1 || result.Fields[0] != "Field5.Field1" {
      t.Errorf("Expected patch result fields to contain exactly \"Field5.Field1\". Contained [%v]", strings.Join(result.Fields, ", "))
      return
    }

    // Test to see if the resulting update map is correct.
    if v, e := result.Map["Field5.Field1"]; !e || v != "test" {
      t.Errorf("Expected patch result map to contain exactly \"Field5.Field1\": \"test\". Contained %v", result.Map)
      return
    }
  })

  t.Run("default-map-patches", func(t *testing.T) {

    testInstance := TestStruct{}

    result, err := Default().Patch(&testInstance, map[string]interface{}{
      "Field6": map[string]int{
        "test1": 255,
      },
    })

    // Test for unexpected errors.
    if err != nil {
      t.Errorf("Unexpected patch error: %q", err.Error())
      return
    }

    // Test to see if the instance's map now exists.
    if testInstance.Field6 == nil {
      t.Errorf("Expected patch to only patch Field5.Field1. Patch affected struct so: %v", testInstance)
      return
    }

    // Test to see if the map contains the expected data.
    if v, e := testInstance.Field6["test1"]; !e || v != 255 {
      t.Errorf("Expected patch to only patch Field6 (map) with \"test\": 255. Patch affected struct so: %v", testInstance)
      return
    }

    // Test to see if the resulting field list is correct.
    if len(result.Fields) != 1 || result.Fields[0] != "Field6" {
      t.Errorf("Expected patch result fields to contain exactly \"Field6\". Contained [%v]", strings.Join(result.Fields, ", "))
      return
    }

    // Test to see if the resulting update map is correct.
    if v, e := result.Map["Field6"]; !e {
      t.Errorf("Expected patch result map to contain exactly \"Field6\": map[string]interface{}{ \"test1\": 255 }. Contained %v (v didn't exist)", result.Map)
      return
    } else if vm, ok := v.(map[string]int); !ok {
      t.Errorf("Expected patch result map to contain exactly \"Field6\": map[string]interface{}{ \"test1\": 255 }. Contained %v (v not map[string]int)", result.Map)
      return
    } else if sv, se := vm["test1"]; !se || sv != 255 {
      t.Errorf("Expected patch result map to contain exactly \"Field6\": map[string]interface{}{ \"test1\": 255 }. Contained %v (sv didn't exist)", result.Map)
      return
    }
  })
}