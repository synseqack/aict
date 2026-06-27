package tool

import (
	"encoding/json"
	"reflect"
	"strings"
)

type Func func(args []string) error

var registry = make(map[string]Func)
var metaRegistry = make(map[string]ToolMeta)

type ToolMeta struct {
	Description string
	InputSchema map[string]interface{}
}

type FlagInfo struct {
	Name        string
	Short       string
	Type        string
	Description string
	Required    bool
}

func Register(name string, fn Func) {
	registry[name] = fn
}

func RegisterMeta(name string, m ToolMeta) {
	metaRegistry[name] = m
}

func All() map[string]Func {
	return registry
}

func AllMeta() map[string]ToolMeta {
	return metaRegistry
}

func GetMeta(name string) (ToolMeta, bool) {
	m, ok := metaRegistry[name]
	return m, ok
}

func MustMarshalJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func GenerateSchema(name string, description string, configPtr interface{}) ToolMeta {
	result := ToolMeta{
		Description: description,
		InputSchema: make(map[string]interface{}),
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
	}

	if configPtr == nil {
		return result
	}

	v := reflect.ValueOf(configPtr)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return result
	}

	t := v.Type()
	props := make(map[string]interface{})
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		flagTag := field.Tag.Get("flag")
		_, hasFlag := field.Tag.Lookup("flag")
		if !hasFlag {
			continue
		}

		fieldName := field.Name
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			fieldName = strings.Split(jsonTag, ",")[0]
		} else {
			fieldName = strings.ToLower(field.Name)
		}

		var fieldType string
		switch field.Type.Kind() {
		case reflect.Bool:
			fieldType = "boolean"
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int8, reflect.Int16:
			fieldType = "integer"
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fieldType = "integer"
		case reflect.String:
			fieldType = "string"
		case reflect.Float64, reflect.Float32:
			fieldType = "number"
		default:
			fieldType = "string"
		}

		desc := field.Tag.Get("desc")
		if desc == "" {
			desc = strings.ToLower(field.Name)
		}

		props[fieldName] = map[string]interface{}{
			"type":        fieldType,
			"description": desc,
		}

		if flagTag == "required" {
			required = append(required, fieldName)
		}
	}

	schema["properties"] = props
	if len(required) > 0 {
		schema["required"] = required
	}

	result.InputSchema = schema
	return result
}
