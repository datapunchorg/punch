package gopatch

// PatchResult is the result of a patch operation. It contains both a list
// of fields patched (the values of which are affected by the Patcher's
// configuration) and a map of the patch performed, prepended with the
// Patcher's configured EmbedPath.
type PatchResult struct {

  // Fields is a lost of field names which were successfully patched. If
  // the Patcher's configured UpdatedFieldSource is empty or  "struct",
  // these names will be sourced from the struct. Otherwise, the Patcher
  // will attempt to find them from the field's tags related to the
  // configured source.
  Fields []string

  // Unpermitted is an array of fields in dot notation which were found to
  // be unpermitted for patching.
  Unpermitted []string

  // Map is a map of the successful patches made to the struct. This is
  // most useful as update data for a corresponding database row or
  // document. When a struct field is encountered and the field's
  // "gopatch" tag doesn't exist, is empty, or is set to "patch", that
  // field's patches are flattened into dot-notation in this map. If the
  // "gopatch" tag's value is set to "replace", the embedded struct will
  // be replaced with the patch data, with any unaccounted-for values
  // being initialized to their zero values.
  Map map[string]interface{}
}