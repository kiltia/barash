package runner

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
)

const (
	QueryTag = "query"
	JsonTag  = "json"
)

// Converts an object to a map of query parameters.
//
// Can work with both concrete and pointer types.
// If the object implements StoredParamsToQuery interface,
// it will be used instead of the manual extraction using tags.
func ObjectToParams(obj any) (
	query url.Values,
) {
	if p, ok := obj.(StoredParamsToQuery); ok {
		query = p.GetQueryParams()
		return query
	}

	query = url.Values{}

	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()

	for i := range val.NumField() {
		field := typ.Field(i)
		queryKey := field.Tag.Get(QueryTag)
		fieldValue := val.Field(i)

		if queryKey != "" && queryKey != "-" {
			if isValueNil(fieldValue) {
				continue
			}
			strRepr := valueToString(fieldValue)
			if strRepr == "" {
				continue
			}
			query.Add(queryKey, strRepr)
		}
	}

	return query
}

// Converts an object to a JSON body.
//
// Can work with both concrete and pointer types.
// If the object implements StoredParamsToBody interface,
// it will be used instead of the manual extraction using tags.
func ObjectToBody(obj any) (
	body []byte,
) {
	if p, ok := obj.(StoredParamsToBody); ok {
		body = p.GetBody()
		return body
	}

	m := map[string]any{}

	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}
	typ := val.Type()

	for i := range val.NumField() {
		field := typ.Field(i)
		jsonKey := field.Tag.Get(JsonTag)
		fieldValue := val.Field(i)

		if jsonKey != "" && jsonKey != "-" {
			m[jsonKey] = fieldValue.Interface()
		}
	}

	bytes, _ := json.Marshal(m)

	return bytes
}

func isValueNil(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice:
		return v.IsNil()
	}
	return false
}

func valueToString(v reflect.Value) string {
	switch v.Kind() {
	case reflect.Pointer:
		return valueToString(v.Elem())
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.String:
		return v.String()
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}
