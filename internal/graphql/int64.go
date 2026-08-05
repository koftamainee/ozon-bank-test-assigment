package graphql

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
)

type Int64 int64

func MarshalInt64(i int64) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		if _, err := io.WriteString(w, strconv.Quote(strconv.FormatInt(i, 10))); err != nil {
			return
		}
	})
}

func UnmarshalInt64(v any) (int64, error) {
	switch v := v.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		if math.Trunc(v) != v {
			return 0, fmt.Errorf("Int64 must be an integer, got %v", v)
		}
		return int64(v), nil
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, fmt.Errorf("Int64 must be an integer: %w", err)
		}
		return n, nil
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("Int64 must be an integer: %w", err)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("Int64 must be an integer, got %T", v)
	}
}
