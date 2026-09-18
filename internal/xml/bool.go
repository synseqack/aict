package xmlout

import (
	"encoding/json"
	"encoding/xml"
)

// compactBools selects 1/0 (true) versus true/false (false) for Bool attributes.
// It is set by WriteXML and WriteXMLNoCompact immediately before marshaling.
// The output path is single-goroutine per tool invocation, so it needs no lock.
var compactBools bool

// Bool is a boolean that marshals to 1/0 as an XML attribute in compact mode and
// to a native JSON boolean, so that boolean encoding never collides with
// attribute values that happen to read "true" or "false".
type Bool bool

func (b Bool) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	value := "false"
	if bool(b) {
		value = "true"
	}
	if compactBools {
		value = "0"
		if bool(b) {
			value = "1"
		}
	}
	return xml.Attr{Name: name, Value: value}, nil
}

func (b Bool) MarshalJSON() ([]byte, error) {
	return json.Marshal(bool(b))
}
