package gopatch

// PatcherConfig is the configuration object used to initialize new
// instances of Patcher.
type PatcherConfig struct {

  // EmbedPath, if set, prepends PatchResult.Map's keys with the specified
  // string in dot notation. For example:
  //
  // // EmbedPath == "profile.metadata"
  //
  // updates, err := gopatch.Default()
  // .Patch(myUser, map[string]interface{}{
  //   "updated_at": "2020-01-01T00:00:00Z",
  // })
  //
  // // updates.Map == map[string]interface{}{
  // //   "profile.metadata.updated_at": "2020-01-01T00:00:00Z",
  // // }
  //
  // This is useful in the case the updated fields and their values should
  // also be applied to a database update operation.
  EmbedPath string

  // PatchSource, defaulting to "struct" when empty, determines from
  // which source a field name comes when matching a patch map's field
  // name to the struct. For "struct", the patch map's keys must match the
  // struct's field names. For any other value, the patcher will match
  // based on the struct fields' matching tags.
  //
  // An empty or "struct" value will use the field's Go-based name.
  // Use of any other value will cause the patcher to search for that tag.
  // Common values include "json", "bson", "msgpack", and "mapstructure"
  PatchSource string

  // PatchErrors causes the Patcher to immediately return an error if a
  // field is encountered which isn't tagged with the one passed to
  // PatchSource, and PatchSource is not empty or "struct". Defaults to
  // false.
  //
  // WARNING: Using this option may result in half-patched structures!
  // Only use this if you don't have further use of the half-patched
  // structure or can reload it afterwards.
  PatchErrors bool

  // UpdatedMapSource, defaulting to "struct" when empty, determines from
  // which source a field name comes when creating the PatchResult.Map
  // map. For "struct", the field string will be the name of the struct
  // field. For any other value, the patcher will look for the related
  // tag and use its value.
  //
  // An empty or "struct" value will use the field's Go-based name.
  // Use of any other value will cause the patcher to search for that tag.
  // Common values include "json", "bson", "msgpack", and "mapstructure"
  UpdatedMapSource string

  // UpdatedMapErrors causes the Patcher to immediately return an error
  // if a field is encountered which isn't tagged with the one passed to
  // UpdatedMapSource, and UpdatedMapSource is not empty or "struct".
  // Defaults to false.
  //
  // WARNING: Using this option may result in half-patched structures!
  // Only use this if you don't have further use of the half-patched
  // structure or can reload it afterwards.
  UpdatedMapErrors bool

  // UpdatedFieldSource, defaulting to "struct" when empty, determines
  // from which source a field name comes when creating the
  // PatchResult.Fields array. For "struct", the field string will be
  // the name of the struct field. For any other value, the patcher
  // will look for the related tag and use its value.
  //
  // An empty or "struct" value will use the field's Go-based name.
  // Use of any other value will cause the patcher to search for that tag.
  // Common values include "json", "bson", "msgpack", and "mapstructure"
  UpdatedFieldSource string

  // UpdatedFieldErrors causes the Patcher to immediately return an error
  // if a field is encountered which isn't tagged with the one passed to
  // UpdatedFieldSource, and UpdatedFieldSource is not empty or "struct".
  // Defaults to false.
  //
  // WARNING: Using this option may result in half-patched structures!
  // Only use this if you don't have further use of the half-patched
  // structure or can reload it afterwards.
  UpdatedFieldErrors bool

  // PermittedFields, if set, will prevent patches from fields in the
  // patch map that are not present in this array. For example:
  //
  // // PermittedFields == []string{"email_address"}
  //
  // updates, err := gopatch.Default()
  // .Patch(myUser, map[string]interface{}{
  //   "email_address": "myemail@address.com",
  //   "password_hash": "injectedhash"
  // })
  //
  // // updates.Fields == []string{"email_address"}
  // // updates.Unpermitted == []string{"password_hash"}
  //
  // To permit all fields of an embedded struct, use `embedded.*`. All
  // fields found to be unpermitted will be stored in dot notation in
  // the PatchResult's UnpermittedFields array if UnpermittedErrors is
  // false.
  PermittedFields []string

  // UnpermittedErrors causes the Patcher to immediately return an error
  // if a field is found to be unpermitted.
  //
  // WARNING: Using this option may result in half-patched structures!
  // Only use this if you don't have further use of the half-patched
  // structure or can reload it afterwards.
  UnpermittedErrors bool
}