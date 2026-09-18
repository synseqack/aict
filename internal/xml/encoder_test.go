package xmlout

import (
	"bytes"
	"encoding/xml"
	"os"
	"strings"
	"testing"
)

type flagProbe struct {
	XMLName xml.Name `xml:"probe"`
	Flag    Bool     `xml:"flag,attr"`
}

// A value that reads "true" must survive compaction as an attribute VALUE.
func TestWriteXML_ValueNamedTrueSurvivesCompaction(t *testing.T) {
	os.Unsetenv("AICT_NOCOMPACT")
	defer os.Unsetenv("AICT_NOCOMPACT")

	type item struct {
		XMLName xml.Name `xml:"item"`
		Name    string   `xml:"name,attr" json:"n"`
	}
	type root struct {
		XMLName xml.Name `xml:"root"`
		Items   []item   `xml:"item"`
	}

	var buf bytes.Buffer
	if err := WriteXML(&buf, root{Items: []item{
		{Name: "true"},
		{Name: "false"},
	}}, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// Attribute NAMES may compact (name -> n, since this fixture has no
	// registered dict and extractAttrMapping derives the short name from the
	// json tag); the VALUES must not. Assert on the value, not the spelling.
	for _, want := range []string{`true"`, `false"`} {
		if !strings.Contains(out, want) {
			t.Errorf("value %q was corrupted: %s", want, out)
		}
	}
	if strings.Contains(out, `="1"`) || strings.Contains(out, `="0"`) {
		t.Errorf("a value was compacted to 1/0: %s", out)
	}
}

// The compact-mode flag must reach MarshalXMLAttr through compactBools.
func TestWriteXML_SetsCompactFlag(t *testing.T) {
	os.Unsetenv("AICT_NOCOMPACT")
	var buf bytes.Buffer
	if err := WriteXML(&buf, &flagProbe{Flag: true}, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `flag="1"`) {
		t.Errorf("compact mode should emit 1, got: %s", buf.String())
	}

	os.Setenv("AICT_NOCOMPACT", "1")
	defer os.Unsetenv("AICT_NOCOMPACT")
	buf.Reset()
	if err := WriteXML(&buf, &flagProbe{Flag: true}, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `flag="true"`) {
		t.Errorf("verbose mode should emit true, got: %s", buf.String())
	}
}

// WriteXMLNoCompact must always emit true/false regardless of env.
func TestWriteXMLNoCompact_AlwaysVerbose(t *testing.T) {
	os.Unsetenv("AICT_NOCOMPACT")
	var buf bytes.Buffer
	if err := WriteXMLNoCompact(&buf, &flagProbe{Flag: true}, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `flag="true"`) {
		t.Errorf("no-compact must emit true, got: %s", buf.String())
	}
}
