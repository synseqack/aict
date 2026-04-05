package xmlout

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

func IsXMLMode() bool {
	return os.Getenv("AICT_XML") == "1"
}

func WriteXML(w io.Writer, v interface{}, pretty bool) error {
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

func WriteJSON(w io.Writer, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte("\n"))
	return err
}

func WriteJSONCompact(w io.Writer, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
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
	if path != "" {
		return fmt.Sprintf("<error code=\"%d\" msg=\"%s\" path=\"%s\"/>", code, msg, path)
	}
	return fmt.Sprintf("<error code=\"%d\" msg=\"%s\"/>", code, msg)
}
