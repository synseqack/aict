package xmlout

import (
	"encoding/json"
	"encoding/xml"
	"strconv"
	"testing"
)

func TestBool_MarshalXMLAttr_Verbose(t *testing.T) {
	compactBools = false
	defer func() { compactBools = false }()

	type s struct {
		XMLName xml.Name `xml:"s"`
		A       Bool     `xml:"a,attr"`
		B       Bool     `xml:"b,attr,omitempty"`
	}

	got, err := xml.Marshal(s{A: true, B: false})
	if err != nil {
		t.Fatal(err)
	}
	// omitempty must still suppress a false Bool.
	want := `<s a="true"></s>`
	if string(got) != want {
		t.Errorf("verbose: got %s, want %s", got, want)
	}
}

func TestBool_MarshalXMLAttr_Compact(t *testing.T) {
	compactBools = true
	defer func() { compactBools = false }()

	type s struct {
		XMLName xml.Name `xml:"s"`
		A       Bool     `xml:"a,attr"`
		C       Bool     `xml:"c,attr"`
	}

	got, err := xml.Marshal(s{A: true, C: false})
	if err != nil {
		t.Fatal(err)
	}
	want := `<s a="1" c="0"></s>`
	if string(got) != want {
		t.Errorf("compact: got %s, want %s", got, want)
	}
}

func TestBool_MarshalJSON(t *testing.T) {
	// JSON is native regardless of compact mode.
	for _, mode := range []bool{true, false} {
		compactBools = mode
		for _, val := range []bool{true, false} {
			got, err := json.Marshal(struct {
				A Bool `json:"a"`
			}{A: Bool(val)})
			if err != nil {
				t.Fatal(err)
			}
			want := `{"a":` + strconv.FormatBool(val) + `}`
			if string(got) != want {
				t.Errorf("compactBools=%v, val=%v: got %s, want %s", mode, val, got, want)
			}
		}
	}
}
