package xmlout

import (
	"encoding/json"
	"encoding/xml"
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
		got, err := json.Marshal(struct {
			A Bool `json:"a"`
		}{A: true})
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != `{"a":true}` {
			t.Errorf("compactBools=%v: got %s, want {\"a\":true}", mode, got)
		}
	}
}
