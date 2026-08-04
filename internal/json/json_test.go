package json

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

type sample struct {
	Name  string   `json:"name"`
	Count int      `json:"count"`
	Tags  []string `json:"tags"`
}

func TestMarshal(t *testing.T) {
	b, err := Marshal(sample{Name: "x", Count: 3, Tags: []string{"a"}})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	want := `{"name":"x","count":3,"tags":["a"]}`
	if string(b) != want {
		t.Errorf("Marshal() = %s, want %s", b, want)
	}
}

func TestMarshalUnsupported(t *testing.T) {
	if _, err := Marshal(func() {}); err == nil {
		t.Fatal("Marshal() = nil, want error for unsupported type")
	}
}

func TestUnmarshalRoundtrip(t *testing.T) {
	in := sample{Name: "hello", Count: 42, Tags: []string{"go", "test"}}
	b, err := Marshal(in)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var out sample
	if err := Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if out.Name != in.Name || out.Count != in.Count || len(out.Tags) != len(in.Tags) || out.Tags[0] != in.Tags[0] {
		t.Errorf("Unmarshal() = %+v, want %+v", out, in)
	}
}

func TestUnmarshalInvalidJSON(t *testing.T) {
	var out sample
	if err := Unmarshal([]byte(`{invalid`), &out); err == nil {
		t.Fatal("Unmarshal() = nil, want error for malformed JSON")
	}
}

func TestEncode(t *testing.T) {
	var buf bytes.Buffer
	if err := Encode(&buf, sample{Name: "n"}); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if got := buf.String(); got != `{"name":"n","count":0,"tags":null}` {
		t.Errorf("Encode() = %s", got)
	}
}

func TestEncodeUnsupported(t *testing.T) {
	var buf bytes.Buffer
	if err := Encode(&buf, func() {}); err == nil {
		t.Fatal("Encode() = nil, want error for unsupported type")
	}
}

func TestDecode(t *testing.T) {
	r := strings.NewReader(`{"name":"d","count":1,"tags":["x"]}`)
	var out sample
	if err := Decode(r, &out); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if out.Name != "d" || out.Count != 1 || len(out.Tags) != 1 {
		t.Errorf("Decode() = %+v", out)
	}
}

func TestDecodeInvalidJSON(t *testing.T) {
	r := strings.NewReader(`not json`)
	var out sample
	if err := Decode(r, &out); err == nil {
		t.Fatal("Decode() = nil, want error for malformed JSON")
	}
}

func TestDecodeReaderError(t *testing.T) {
	r := &errReader{err: errors.New("read failed")}
	var out sample
	if err := Decode(r, &out); err == nil {
		t.Fatal("Decode() = nil, want reader error")
	}
}

type errReader struct {
	err error
}

func (r *errReader) Read(_ []byte) (int, error) {
	return 0, r.err
}
