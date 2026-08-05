package config

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
	"unicode"
)

type DefaultConfig struct {
	Env string `default:"dev"`
}

func (c *DefaultConfig) GetEnv() string {
	return c.Env
}

func MustLoad(dst any) {
	if err := Load(dst); err != nil {
		panic(err)
	}
}

func Load(dst any) error {
	if err := load(reflect.ValueOf(dst).Elem(), ""); err != nil {
		return err
	}

	if dc, ok := dst.(interface{ GetEnv() string }); ok {
		env := dc.GetEnv()
		if env != "dev" && env != "prod" {
			return fmt.Errorf("env must be 'dev' or 'prod', got '%s'", env)
		}
	}

	return nil
}

func load(v reflect.Value, prefix string) error {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldVal := v.Field(i)

		if !field.IsExported() {
			continue
		}

		key := buildKey(prefix, field.Name)
		defaultVal := field.Tag.Get("default")

		if fieldVal.Kind() == reflect.Struct {
			if err := load(fieldVal, key); err != nil {
				return err
			}
			continue
		}

		raw, ok := readSource(key)
		if ok {
			if err := setField(fieldVal, raw); err != nil {
				return fmt.Errorf("field %s: %w", key, err)
			}
			continue
		}

		if defaultVal != "" {
			if err := setField(fieldVal, defaultVal); err != nil {
				return fmt.Errorf("field %s (default): %w", key, err)
			}
			continue
		}

		if field.Tag.Get("optional") == "true" {
			continue
		}

		return fmt.Errorf("field %s: no value and no default", key)
	}

	return nil
}

func buildKey(prefix, name string) string {
	key := camelToSnake(name)
	if prefix != "" {
		return prefix + "." + key
	}
	return key
}

func camelToSnake(s string) string {
	runes := []rune(s)
	var result []rune
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			if unicode.IsLower(prev) {
				result = append(result, '_')
			} else if unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
				result = append(result, '_')
			}
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

func setField(v reflect.Value, raw string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(raw)
	case reflect.Int, reflect.Int32, reflect.Int64:
		if v.Type() == reflect.TypeFor[time.Duration]() {
			d, err := time.ParseDuration(raw)
			if err != nil {
				return err
			}
			v.SetInt(int64(d))
		} else {
			n, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return err
			}
			v.SetInt(n)
		}
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		v.SetBool(b)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		v.SetFloat(f)
	default:
		return fmt.Errorf("unsupported type: %s", v.Kind())
	}
	return nil
}
