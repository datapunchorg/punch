package gopatch

import (
  "database/sql"
  "reflect"
  "time"

  "github.com/guregu/null"
)

// MapUpdater updates any map as long as the key and element values match.
func MapUpdater(fieldValue reflect.Value, v reflect.Value) bool {
  if fieldValue.Kind() != reflect.Map { return false }
  if fieldValue.Type().Key().Kind() != v.Type().Key().Kind() { return false }
  if fieldValue.Type().Elem().Kind() != v.Type().Elem().Kind() {
    if !v.Type().ConvertibleTo(fieldValue.Type()) { return false }

    fieldValue.Set(v.Convert(fieldValue.Type()))
  }
  
  fieldValue.Set(v)
  return true
}

// NullStringUpdater updates null.String
func NullStringUpdater(fieldValue reflect.Value, v reflect.Value) bool {
  switch fieldValue.Interface().(type) {
  case null.String:
    // if its null value
    if !v.IsValid() {
      newValue := reflect.ValueOf(null.String{NullString: sql.NullString{Valid: false}})
      fieldValue.Set(newValue)
      return true
    }
    // only set if underlying type is string
    if v.Kind() == reflect.String {
      newValue := reflect.ValueOf(null.String{NullString: sql.NullString{Valid: true, String: v.String()}})
      fieldValue.Set(newValue)
      return true
    }
  }

  return false
}

// NullFloatUpdater updates null.Float64
func NullFloatUpdater(fieldValue reflect.Value, v reflect.Value) bool {
  switch fieldValue.Interface().(type) {
  case null.Float:
    // if its null value
    if !v.IsValid() {
      newValue := reflect.ValueOf(null.Float{NullFloat64: sql.NullFloat64{Valid: false}})
      fieldValue.Set(newValue)
      return true
    }
    // only set if underlying type is any int/float
    if v.Kind() == reflect.Int ||
      v.Kind() == reflect.Int8 ||
      v.Kind() == reflect.Int16 ||
      v.Kind() == reflect.Int32 ||
      v.Kind() == reflect.Int64 {
      newValue := reflect.ValueOf(null.Float{NullFloat64: sql.NullFloat64{Valid: true, Float64: float64(v.Int())}})
      fieldValue.Set(newValue)
      return true
    } else if v.Kind() == reflect.Float32 ||
      v.Kind() == reflect.Float64 {
      newValue := reflect.ValueOf(null.Float{NullFloat64: sql.NullFloat64{Valid: true, Float64: v.Float()}})
      fieldValue.Set(newValue)
      return true
    }
  }

  return false
}

// NullIntUpdater updates null.Int
func NullIntUpdater(fieldValue reflect.Value, v reflect.Value) bool {
  switch fieldValue.Interface().(type) {
  case null.Int:
    // if its null value
    if !v.IsValid() {
      newValue := reflect.ValueOf(null.Int{NullInt64: sql.NullInt64{Valid: false}})
      fieldValue.Set(newValue)
      return true
    }
    // only set if underlying type is any int/float
    if v.Kind() == reflect.Int ||
      v.Kind() == reflect.Int8 ||
      v.Kind() == reflect.Int16 ||
      v.Kind() == reflect.Int32 ||
      v.Kind() == reflect.Int64 {
      newValue := reflect.ValueOf(null.Int{NullInt64: sql.NullInt64{Valid: true, Int64: v.Int()}})
      fieldValue.Set(newValue)
      return true
    } else if v.Kind() == reflect.Float32 ||
      v.Kind() == reflect.Float64 {
      newValue := reflect.ValueOf(null.Int{NullInt64: sql.NullInt64{Valid: true, Int64: int64(v.Float())}})
      fieldValue.Set(newValue)
      return true
    }
  }

  return false
}

// NullBoolUpdater updates null.Bool
func NullBoolUpdater(fieldValue reflect.Value, v reflect.Value) bool {
  switch fieldValue.Interface().(type) {
  case null.Bool:
    // if its null value
    if !v.IsValid() {
      newValue := reflect.ValueOf(null.Bool{NullBool: sql.NullBool{Valid: false}})
      fieldValue.Set(newValue)
      return true
    }
    // only set if underlying type is bool
    if v.Kind() == reflect.Bool {
      newValue := reflect.ValueOf(null.Bool{NullBool: sql.NullBool{Valid: true, Bool: v.Bool()}})
      fieldValue.Set(newValue)
      return true
    }
  }

  return false
}

// NullTimeUpdater updates null.Time
func NullTimeUpdater(fieldValue reflect.Value, v reflect.Value) bool {
  switch fieldValue.Interface().(type) {
  case null.Time:
    // if its null value
    if !v.IsValid() {
      newValue := reflect.ValueOf(null.Time{Valid: false})
      fieldValue.Set(newValue)
      return true
    }
    // only set if underlying type is string
    if v.Kind() == reflect.String {
      nullTime := null.Time{}
      if err := nullTime.UnmarshalJSON([]byte(`"` + v.String() + `"`)); err == nil {
        newValue := reflect.ValueOf(nullTime)
        fieldValue.Set(newValue)
        return true
      }
    }
  }

  return false
}

// BoolUpdater updates bool (pointer or value)
func BoolUpdater(fieldValue reflect.Value, v reflect.Value) bool {
  if fieldValue.Kind() == reflect.Bool {
    if v.Kind() == reflect.Bool {
      fieldValue.Set(v)
      return true
    }
  } else if fieldValue.Kind() == reflect.Ptr {
    // only process if field is pointer to any bool
    if fieldValue.Type().String() == "*bool" {
      if !v.IsValid() {
        var newBoolValue *bool
        newValue := reflect.ValueOf(newBoolValue)
        fieldValue.Set(newValue)
        return true
      } else if v.Kind() == reflect.Bool {
        newBoolValue := v.Bool()
        newValue := reflect.ValueOf(&newBoolValue)
        fieldValue.Set(newValue)
        return true
      }
    }
  }

  return false
}

// IntUpdater updates int (any int type Int8, Int16, Int32, Int64 and whether its a pointer or a value)
func IntUpdater(fieldValue reflect.Value, v reflect.Value) bool {
  if fieldValue.Kind() == reflect.Int ||
    fieldValue.Kind() == reflect.Int8 ||
    fieldValue.Kind() == reflect.Int16 ||
    fieldValue.Kind() == reflect.Int32 ||
    fieldValue.Kind() == reflect.Int64 {

    if v.Kind() == reflect.Int ||
      v.Kind() == reflect.Int8 ||
      v.Kind() == reflect.Int16 ||
      v.Kind() == reflect.Int32 ||
      v.Kind() == reflect.Int64 {
      fieldValue.SetInt(v.Int())
      return true
    } else if v.Kind() == reflect.Float32 ||
      v.Kind() == reflect.Float64 {
      fieldValue.SetInt(int64(v.Float()))
      return true
    }
  } else if fieldValue.Kind() == reflect.Ptr {
    // only process if field is pointer to any int
    if fieldValue.Type().String() == "*int" ||
      fieldValue.Type().String() == "*int8" ||
      fieldValue.Type().String() == "*int16" ||
      fieldValue.Type().String() == "*int32" ||
      fieldValue.Type().String() == "*int64" {
      if !v.IsValid() {
        if fieldValue.Type().String() == "*int" {
          var newNullInt *int
          newValue := reflect.ValueOf(newNullInt)
          fieldValue.Set(newValue)
          return true
        } else if fieldValue.Type().String() == "*int8" {
          var newNullInt8 *int8
          newValue := reflect.ValueOf(newNullInt8)
          fieldValue.Set(newValue)
          return true
        } else if fieldValue.Type().String() == "*int16" {
          var newNullInt16 *int16
          newValue := reflect.ValueOf(newNullInt16)
          fieldValue.Set(newValue)
          return true
        } else if fieldValue.Type().String() == "*int32" {
          var newNullInt32 *int32
          newValue := reflect.ValueOf(newNullInt32)
          fieldValue.Set(newValue)
          return true
        } else if fieldValue.Type().String() == "*int64" {
          var newNullInt64 *int64
          newValue := reflect.ValueOf(newNullInt64)
          fieldValue.Set(newValue)
          return true
        }
      } else if v.Kind() == reflect.Int ||
        v.Kind() == reflect.Int8 ||
        v.Kind() == reflect.Int16 ||
        v.Kind() == reflect.Int32 ||
        v.Kind() == reflect.Int64 {
        if fieldValue.Type().String() == "*int" {
          newIntValue := int(v.Int())
          newValue := reflect.ValueOf(&newIntValue)
          fieldValue.Set(newValue)
          return true
        } else if fieldValue.Type().String() == "*int8" {
          newInt8Value := int8(v.Int())
          newValue := reflect.ValueOf(&newInt8Value)
          fieldValue.Set(newValue)
          return true
        } else if fieldValue.Type().String() == "*int16" {
          newInt16Value := int16(v.Int())
          newValue := reflect.ValueOf(&newInt16Value)
          fieldValue.Set(newValue)
          return true
        } else if fieldValue.Type().String() == "*int32" {
          newInt32Value := int32(v.Int())
          newValue := reflect.ValueOf(&newInt32Value)
          fieldValue.Set(newValue)
          return true
        } else if fieldValue.Type().String() == "*int64" {
          newInt64Value := v.Int()
          newValue := reflect.ValueOf(&newInt64Value)
          fieldValue.Set(newValue)
          return true
        }
      } else if v.Kind() == reflect.Float32 ||
        v.Kind() == reflect.Float64 {
        if fieldValue.Type().String() == "*int" {
          newIntValue := int(v.Float())
          newValue := reflect.ValueOf(&newIntValue)
          fieldValue.Set(newValue)
          return true
        } else if fieldValue.Type().String() == "*int8" {
          newInt8Value := int8(v.Float())
          newValue := reflect.ValueOf(&newInt8Value)
          fieldValue.Set(newValue)
          return true
        } else if fieldValue.Type().String() == "*int16" {
          newInt16Value := int16(v.Float())
          newValue := reflect.ValueOf(&newInt16Value)
          fieldValue.Set(newValue)
          return true
        } else if fieldValue.Type().String() == "*int32" {
          newInt32Value := int32(v.Float())
          newValue := reflect.ValueOf(&newInt32Value)
          fieldValue.Set(newValue)
          return true
        } else if fieldValue.Type().String() == "*int64" {
          newInt64Value := int64(v.Float())
          newValue := reflect.ValueOf(&newInt64Value)
          fieldValue.Set(newValue)
          return true
        }
      }
    }
  }

  return false
}

// FloatUpdater updates int (any float type Float8, Float16, Float32, Float64 and whether its a pointer or a value)
func FloatUpdater(fieldValue reflect.Value, v reflect.Value) bool {
  if fieldValue.Kind() == reflect.Float32 ||
    fieldValue.Kind() == reflect.Float64 {

    if v.Kind() == reflect.Int ||
      v.Kind() == reflect.Int8 ||
      v.Kind() == reflect.Int16 ||
      v.Kind() == reflect.Int32 ||
      v.Kind() == reflect.Int64 {
      fieldValue.SetFloat(float64(v.Int()))
      return true
    } else if v.Kind() == reflect.Float32 ||
      v.Kind() == reflect.Float64 {
      fieldValue.SetFloat(v.Float())
      return true
    }
  } else if fieldValue.Kind() == reflect.Ptr {
    // only process if field is pointer to any float
    if fieldValue.Type().String() == "*float32" ||
      fieldValue.Type().String() == "*float64" {
      if !v.IsValid() {
        if fieldValue.Type().String() == "*float32" {
          var newFloat32Value *float32
          newValue := reflect.ValueOf(newFloat32Value)
          fieldValue.Set(newValue)
          return true
        } else if fieldValue.Type().String() == "*float64" {
          var newFloat64Value *float64
          newValue := reflect.ValueOf(newFloat64Value)
          fieldValue.Set(newValue)
          return true
        }
      } else if v.Kind() == reflect.Int ||
        v.Kind() == reflect.Int8 ||
        v.Kind() == reflect.Int16 ||
        v.Kind() == reflect.Int32 ||
        v.Kind() == reflect.Int64 {
        if fieldValue.Type().String() == "*float32" {
          newFloat32Value := float32(v.Int())
          newValue := reflect.ValueOf(&newFloat32Value)
          fieldValue.Set(newValue)
          return true
        } else if fieldValue.Type().String() == "*float64" {
          newFloat64Value := float64(v.Int())
          newValue := reflect.ValueOf(&newFloat64Value)
          fieldValue.Set(newValue)
          return true
        }
      } else if v.Kind() == reflect.Float32 ||
        v.Kind() == reflect.Float64 {
        if fieldValue.Type().String() == "*float32" {
          newFloat32Value := float32(v.Float())
          newValue := reflect.ValueOf(&newFloat32Value)
          fieldValue.Set(newValue)
          return true
        } else if fieldValue.Type().String() == "*float64" {
          newFloat64Value := v.Float()
          newValue := reflect.ValueOf(&newFloat64Value)
          fieldValue.Set(newValue)
          return true
        }
      }
    }
  }

  return false
}

// TimeUpdater updates time (pointer or value)
func TimeUpdater(fieldValue reflect.Value, v reflect.Value) bool {
  switch fieldValue.Interface().(type) {
  case time.Time:
    if !v.IsValid() {
      return false
    }
    // only set if underlying type is string
    if v.Kind() == reflect.String {
      t := time.Now()
      // make sure date format is correct
      if err := t.UnmarshalJSON([]byte(`"` + v.String() + `"`)); err == nil {
        newValue := reflect.ValueOf(t)
        fieldValue.Set(newValue)
        return true
      }
    }
  case *time.Time:
    if !v.IsValid() {
      var t *time.Time
      newValue := reflect.ValueOf(t)
      fieldValue.Set(newValue)
      return true
    }
    // only set if underlying type is string
    if v.Kind() == reflect.String {
      t := time.Now()
      // make sure date format is correct
      if err := t.UnmarshalJSON([]byte(`"` + v.String() + `"`)); err == nil {
        newValue := reflect.ValueOf(&t)
        fieldValue.Set(newValue)
        return true
      }
    }
  }

  return false
}

// Updaters is a collection of all default type updaters.
var Updaters = []func(reflect.Value, reflect.Value) bool{
  NullStringUpdater,
  NullFloatUpdater,
  NullIntUpdater,
  NullBoolUpdater,
  NullTimeUpdater,
  MapUpdater,
  IntUpdater,
  FloatUpdater,
  TimeUpdater,
  BoolUpdater,
}