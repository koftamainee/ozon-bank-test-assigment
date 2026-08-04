package json

import (
	"io"

	"github.com/bytedance/sonic"
)

func Marshal(v any) ([]byte, error) {
	return sonic.Marshal(v)
}

func Unmarshal(data []byte, v any) error {
	return sonic.Unmarshal(data, v)
}

func Encode(w io.Writer, v any) error {
	b, err := sonic.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func Decode(r io.Reader, v any) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	return sonic.Unmarshal(b, v)
}
