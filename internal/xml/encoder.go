package xmlout

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
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
	type errElem struct {
		XMLName xml.Name `xml:"error"`
		Code    int      `xml:"code,attr"`
		Msg     string   `xml:"msg,attr"`
		Path    string   `xml:"path,attr,omitempty"`
	}
	data, _ := xml.Marshal(errElem{Code: code, Msg: msg, Path: path})
	return string(data)
}

