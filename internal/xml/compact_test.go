package xmlout

import (
	"bytes"
	"encoding/xml"
	"os"
	"regexp"
	"strings"
	"testing"
)

type testResult struct {
	XMLName      xml.Name    `xml:"test" json:"-"`
	Path         string      `xml:"path,attr" json:"p"`
	Absolute     string      `xml:"absolute,attr" json:"a"`
	TotalEntries int         `xml:"total_entries,attr" json:"n"`
	Hidden       bool        `xml:"hidden,attr" json:"h"`
	Timestamp    int64       `xml:"timestamp,attr" json:"t"`
	Entries      []testEntry `xml:"entry,omitempty" json:"entries,omitempty"`
}

type testEntry struct {
	XMLName   xml.Name `xml:"entry" json:"-"`
	Name      string   `xml:"name,attr" json:"nm"`
	SizeBytes int64    `xml:"size_bytes,attr" json:"s"`
	SizeHuman string   `xml:"size_human,attr" json:"sh"`
	Language  string   `xml:"language,attr" json:"lang"`
	Binary    bool     `xml:"binary,attr" json:"bin"`
}

func init() {
	RegisterDict("test", map[string]string{
		"p":    "path",
		"a":    "absolute",
		"n":    "total_entries",
		"h":    "hidden",
		"nm":   "name",
		"s":    "size_bytes",
		"sh":   "size_human",
		"lang": "language",
		"bin":  "binary",
	})
}

func TestCompactXML_ReplacesAttributes(t *testing.T) {
	os.Unsetenv("AICT_NOCOMPACT")

	result := &testResult{
		Path:         ".",
		Absolute:     "/home/user/project",
		TotalEntries: 2,
		Hidden:       false,
		Timestamp:    1700000000,
		Entries: []testEntry{
			{Name: "main.go", SizeBytes: 1024, SizeHuman: "1K", Language: "go", Binary: false},
			{Name: "image.png", SizeBytes: 5120, SizeHuman: "5K", Language: "", Binary: true},
		},
	}

	var buf bytes.Buffer
	err := WriteXML(&buf, result, false)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// Should use short attribute names
	if !strings.Contains(output, `p="."`) {
		t.Errorf("expected short attr 'p', got: %s", output)
	}
	if !strings.Contains(output, `a="/home/user/project"`) {
		t.Errorf("expected short attr 'a', got: %s", output)
	}
	if !strings.Contains(output, `n="2"`) {
		t.Errorf("expected short attr 'n', got: %s", output)
	}

	// Should NOT use long attribute names
	if strings.Contains(output, `path="."`) {
		t.Errorf("should not contain long attr 'path', got: %s", output)
	}
	if strings.Contains(output, `absolute="`) {
		t.Errorf("should not contain long attr 'absolute', got: %s", output)
	}
	if strings.Contains(output, `total_entries="`) {
		t.Errorf("should not contain long attr 'total_entries', got: %s", output)
	}

	// Booleans should be 1/0
	if strings.Contains(output, `="true"`) || strings.Contains(output, `="false"`) {
		t.Errorf("booleans should be 1/0, got: %s", output)
	}
	if !strings.Contains(output, `h="0"`) {
		t.Errorf("expected h='0' for false, got: %s", output)
	}
}

func TestCompactXML_NestedEntries(t *testing.T) {
	os.Unsetenv("AICT_NOCOMPACT")

	result := &testResult{
		Path:         ".",
		Absolute:     "/tmp",
		TotalEntries: 1,
		Entries: []testEntry{
			{Name: "file.txt", SizeBytes: 100, SizeHuman: "100B", Language: "text", Binary: false},
		},
	}

	var buf bytes.Buffer
	err := WriteXML(&buf, result, false)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// Entry should also use short names
	if !strings.Contains(output, `nm="file.txt"`) {
		t.Errorf("expected short attr 'nm' for entry name, got: %s", output)
	}
	if !strings.Contains(output, `s="100"`) {
		t.Errorf("expected short attr 's' for size_bytes, got: %s", output)
	}
	if !strings.Contains(output, `lang="text"`) {
		t.Errorf("expected short attr 'lang' for language, got: %s", output)
	}
	if !strings.Contains(output, `bin="0"`) {
		t.Errorf("expected short attr 'bin' for binary=false, got: %s", output)
	}
}

func TestCompactXML_ValidXML(t *testing.T) {
	os.Unsetenv("AICT_NOCOMPACT")

	result := &testResult{
		Path:         ".",
		Absolute:     "/tmp",
		TotalEntries: 1,
		Timestamp:    1700000000,
		Entries: []testEntry{
			{Name: "test.go", SizeBytes: 50, SizeHuman: "50B", Language: "go"},
		},
	}

	var buf bytes.Buffer
	err := WriteXML(&buf, result, false)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// Must be valid XML
	var parsed testResult
	if err := xml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("compact XML is not valid: %v\nOutput: %s", err, output)
	}
}

func TestCompactXML_NoCompactMode(t *testing.T) {
	os.Setenv("AICT_NOCOMPACT", "1")
	defer os.Unsetenv("AICT_NOCOMPACT")

	result := &testResult{
		Path:         ".",
		Absolute:     "/tmp",
		TotalEntries: 1,
		Timestamp:    1700000000,
	}

	var buf bytes.Buffer
	err := WriteXML(&buf, result, false)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// Should use long attribute names
	if !strings.Contains(output, `path="."`) {
		t.Errorf("expected long attr 'path' in no-compact mode, got: %s", output[:200])
	}
	if !strings.Contains(output, `absolute="/tmp"`) {
		t.Errorf("expected long attr 'absolute' in no-compact mode, got: %s", output[:200])
	}
}

func TestGetRegisteredDict(t *testing.T) {
	dict := GetRegisteredDict("test")
	if dict == nil {
		t.Fatal("expected non-nil dict for 'test'")
	}

	if dict["p"] != "path" {
		t.Errorf("expected p -> path, got %s", dict["p"])
	}
	if dict["a"] != "absolute" {
		t.Errorf("expected a -> absolute, got %s", dict["a"])
	}
	if dict["s"] != "size_bytes" {
		t.Errorf("expected s -> size_bytes, got %s", dict["s"])
	}
}

func TestGetRegisteredDict_Unknown(t *testing.T) {
	dict := GetRegisteredDict("nonexistent_tool")
	if dict != nil {
		t.Errorf("expected nil for unknown tool, got %v", dict)
	}
}

func TestGetDictXML(t *testing.T) {
	result := &testResult{}
	dictXML := GetDictXML("test", result)

	if !strings.HasPrefix(dictXML, "<dict>") {
		t.Errorf("expected <dict> prefix, got: %s", dictXML)
	}
	if !strings.HasSuffix(dictXML, "</dict>") {
		t.Errorf("expected </dict> suffix, got: %s", dictXML)
	}

	// Should contain the mappings
	if !strings.Contains(dictXML, "<p>path</p>") {
		t.Errorf("expected <p>path</p> in dict, got: %s", dictXML)
	}
	if !strings.Contains(dictXML, "<a>absolute</a>") {
		t.Errorf("expected <a>absolute</a> in dict, got: %s", dictXML)
	}

	// Should be sorted by short name
	idxP := strings.Index(dictXML, "<p>")
	idxA := strings.Index(dictXML, "<a>")
	if idxA > idxP {
		t.Errorf("dict should be sorted alphabetically, <a> should come before <p>")
	}
}

func TestGetJSONDict(t *testing.T) {
	result := &testResult{}
	jsonDict := GetJSONDict("test", result)

	if !strings.HasPrefix(jsonDict, "{") || !strings.HasSuffix(jsonDict, "}") {
		t.Errorf("expected JSON object, got: %s", jsonDict)
	}

	if !strings.Contains(jsonDict, `"p":"path"`) {
		t.Errorf("expected p:path in JSON dict, got: %s", jsonDict)
	}
}

func TestWriteJSON_CompactKeys(t *testing.T) {
	result := &testResult{
		Path:         ".",
		Absolute:     "/tmp",
		TotalEntries: 2,
		Hidden:       false,
		Timestamp:    1700000000,
	}

	var buf bytes.Buffer
	err := WriteJSON(&buf, result)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// Should use short keys
	if !strings.Contains(output, `"p"`) {
		t.Errorf("expected short key 'p', got: %s", output)
	}
	if !strings.Contains(output, `"a"`) {
		t.Errorf("expected short key 'a', got: %s", output)
	}

	// Booleans should be 1/0
	if strings.Contains(output, `:true`) || strings.Contains(output, `:false`) {
		t.Errorf("booleans should be 1/0 in JSON, got: %s", output)
	}
}

func TestCompactXML_TagNamesPreserved(t *testing.T) {
	os.Unsetenv("AICT_NOCOMPACT")

	result := &testResult{
		Path:         ".",
		Absolute:     "/tmp",
		TotalEntries: 1,
		Timestamp:    1700000000,
		Entries: []testEntry{
			{Name: "f", SizeBytes: 1, SizeHuman: "1B", Language: "go"},
		},
	}

	var buf bytes.Buffer
	err := WriteXML(&buf, result, false)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// XML element names should NOT be compacted (only attributes)
	if !strings.Contains(output, "<test ") {
		t.Errorf("root element name 'test' should be preserved, got: %s", output[:200])
	}
	if !strings.Contains(output, "<entry ") {
		t.Errorf("child element name 'entry' should be preserved, got: %s", output[:200])
	}
}

func TestCompactXML_EmptyEntries(t *testing.T) {
	os.Unsetenv("AICT_NOCOMPACT")

	result := &testResult{
		Path:         ".",
		Absolute:     "/tmp",
		TotalEntries: 0,
		Timestamp:    1700000000,
		Entries:      nil,
	}

	var buf bytes.Buffer
	err := WriteXML(&buf, result, false)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// Empty entries should still produce valid compact XML
	if !strings.Contains(output, `n="0"`) {
		t.Errorf("expected n='0' for empty result, got: %s", output)
	}

	var parsed testResult
	if err := xml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("empty compact XML is not valid: %v\nOutput: %s", err, output)
	}
}

func TestCompactXML_Feature(t *testing.T) {
	os.Unsetenv("AICT_NOCOMPACT")

	// Test that the 'feature' (f) element name is preserved
	result := &testResult{
		Path:         ".",
		Absolute:     "/tmp",
		TotalEntries: 1,
		Timestamp:    1700000000,
		Entries: []testEntry{
			{Name: "test", SizeBytes: 10, SizeHuman: "10B", Language: "text", Binary: false},
		},
	}

	var buf bytes.Buffer
	err := WriteXML(&buf, result, false)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// Verify compact output is shorter than verbose
	os.Setenv("AICT_NOCOMPACT", "1")
	var bufVerbose bytes.Buffer
	_ = WriteXML(&bufVerbose, result, false)
	os.Unsetenv("AICT_NOCOMPACT")

	verbose := bufVerbose.String()
	compact := output

	if len(compact) >= len(verbose) {
		t.Errorf("compact (%d bytes) should be shorter than verbose (%d bytes)", len(compact), len(verbose))
	}
}

func TestCompactXML_AttributeRegex(t *testing.T) {
	// Test the regex pattern used for attribute replacement
	pattern := regexp.MustCompile(`([ >]|<[a-z]+ )([a-z_]+)="`)

	tests := []struct {
		input    string
		expected string
	}{
		{` <entry name="test"`, ` <entry nm="test"`},
		{`> name="test"`, `> nm="test"`},
		{` name="test"`, ` nm="test"`},
	}

	longToShort := map[string]string{"name": "nm"}

	for _, tt := range tests {
		result := pattern.ReplaceAllStringFunc(tt.input, func(match string) string {
			parts := pattern.FindStringSubmatch(match)
			if len(parts) < 3 {
				return match
			}
			prefix := parts[1]
			attrName := parts[2]
			if short, ok := longToShort[attrName]; ok {
				return prefix + short + `="`
			}
			return match
		})

		if result != tt.expected {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}
