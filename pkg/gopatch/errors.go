package gopatch

import(
  "errors"
)

var errDestInvalid = errors.New("dest interface invalid, must be non-nil pointer to struct")

func errFieldMissingTag(field, tag string) error { return errors.New("field `"+field+"` is missing tag `"+tag+"`")}
func errFieldUnpermitted(field, cause string) error { return errors.New("field `"+field+"` is not permitted due to `"+cause+"`")}