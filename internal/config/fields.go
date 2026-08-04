package config

import "reflect"

type FieldInfo struct {
	Key        string
	HasDefault bool
	Default    string
	SourceKey  string
}

func Fields(dst any) []FieldInfo {
	v := reflect.ValueOf(dst).Elem()
	t := v.Type()
	var fields []FieldInfo
	collectFields(v, t, "", &fields)
	return fields
}

func collectFields(v reflect.Value, t reflect.Type, prefix string, fields *[]FieldInfo) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)

		if !field.IsExported() {
			continue
		}

		key := buildKey(prefix, field.Name)

		if fieldVal.Kind() == reflect.Struct {
			collectFields(fieldVal, field.Type, key, fields)
			continue
		}

		defaultVal := field.Tag.Get("default")

		*fields = append(*fields, FieldInfo{
			Key:        key,
			HasDefault: defaultVal != "",
			Default:    defaultVal,
			SourceKey:  formatKey(key),
		})
	}
}
