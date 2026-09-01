package xmlout

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func IsXMLMode() bool {
	return os.Getenv("AICT_XML") == "1"
}

// WriteXML writes XML with short attribute names (compact by default)
func WriteXML(w io.Writer, v interface{}, pretty bool) error {
	data, err := xml.Marshal(v)
	if err != nil {
		return err
	}

	toolName := detectToolName(v)
	compact := compactXML(string(data), v, toolName)
	_, err = w.Write([]byte(compact))
	return err
}

// WriteXMLNoCompact writes XML with long attribute names (verbose, for backward compatibility)
func WriteXMLNoCompact(w io.Writer, v interface{}, pretty bool) error {
	if pretty {
		data, err := xml.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	enc := xml.NewEncoder(w)
	return enc.Encode(v)
}

func WriteXMLStream(w io.Writer, elementName string, items []string) error {
	enc := xml.NewEncoder(w)

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	_, err := fmt.Fprintf(w, "<%s timestamp=\"%s\">", elementName, ts)
	if err != nil {
		return err
	}

	for _, item := range items {
		err = enc.EncodeToken(xml.CharData(item))
		if err != nil {
			return err
		}
	}

	err = enc.EncodeToken(xml.CharData([]byte("\n")))
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "</%s>", elementName)
	return err
}

// WriteJSON writes JSON with short keys (always compact)
func WriteJSON(w io.Writer, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	toolName := detectToolName(v)
	compact := compactJSONKeys(string(data), v, toolName)
	_, err = w.Write([]byte(compact))
	if err != nil {
		return err
	}
	_, err = w.Write([]byte("\n"))
	return err
}

// WriteJSONCompact writes compact JSON without pretty printing
func WriteJSONCompact(w io.Writer, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	toolName := detectToolName(v)
	compact := compactJSONKeys(string(data), v, toolName)
	_, err = w.Write([]byte(compact))
	if err != nil {
		return err
	}
	_, err = w.Write([]byte("\n"))
	return err
}

func WritePlain(w io.Writer, formatFn func(io.Writer, interface{}) error, v interface{}) error {
	if formatFn == nil {
		return fmt.Errorf("no plain text formatter provided")
	}
	return formatFn(w, v)
}

func ErrorElement(code int, msg string, path string) string {
	type errElem struct {
		XMLName xml.Name `xml:"error"`
		Code    int      `xml:"code,attr"`
		Msg     string   `xml:"msg,attr"`
		Path    string   `xml:"path,attr,omitempty"`
	}
	data, _ := xml.Marshal(errElem{Code: code, Msg: msg, Path: path})
	return string(data)
}

// detectToolName auto-detects the tool name from the struct's XML root element
func detectToolName(v interface{}) string {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return ""
	}

	// Check for XMLName field with non-empty Local
	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		if field.Name == "XMLName" {
			fieldVal := val.Field(i)
			if fieldVal.Kind() == reflect.Struct {
				local := fieldVal.FieldByName("Local")
				if local.IsValid() && local.Kind() == reflect.String && local.String() != "" {
					return local.String()
				}
			}
		}
	}

	// Check struct type name (e.g., LSResult -> ls)
	typeName := val.Type().Name()
	if strings.HasSuffix(typeName, "Result") {
		return strings.ToLower(strings.TrimSuffix(typeName, "Result"))
	}

	return ""
}
