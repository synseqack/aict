package xmlout

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
)

// GetDictXML returns the dictionary for a tool by reading struct tags
func GetDictXML(toolName string, v interface{}) string {
	dict := make(map[string]string)

	// Extract mappings from struct tags
	extractDictFromStruct(v, dict)

	// Sort by short name for deterministic output
	var keys []string
	for short := range dict {
		keys = append(keys, short)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("<dict>")
	for _, short := range keys {
		long := dict[short]
		sb.WriteString("<" + short + ">" + long + "</" + short + ">")
	}
	sb.WriteString("</dict>")
	return sb.String()
}

// registeredDicts holds pre-registered dictionaries for tools
var registeredDicts = map[string]map[string]string{}

// RegisterDict registers a dictionary for a tool
func RegisterDict(toolName string, dict map[string]string) {
	registeredDicts[toolName] = dict
}

// GetRegisteredDict returns a registered dictionary
func GetRegisteredDict(toolName string) map[string]string {
	if dict, ok := registeredDicts[toolName]; ok {
		return dict
	}
	return nil
}

// extractDictFromStruct reads struct tags to build short->long mapping
func extractDictFromStruct(v interface{}, dict map[string]string) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Get XML tag
		xmlTag := field.Tag.Get("xml")
		if xmlTag == "" || xmlTag == "-" {
			continue
		}

		// Parse XML tag: "name,attr" or "name"
		xmlParts := strings.Split(xmlTag, ",")
		xmlName := xmlParts[0]
		if xmlName == "" {
			continue
		}

		// Get JSON tag for the short name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		jsonParts := strings.Split(jsonTag, ",")
		shortName := jsonParts[0]
		if shortName == "" {
			continue
		}

		// Add to dictionary
		dict[shortName] = xmlName

		// If field is a struct or slice of structs, recurse
		if fieldVal.Kind() == reflect.Ptr {
			fieldVal = fieldVal.Elem()
		}
		if fieldVal.Kind() == reflect.Struct {
			extractDictFromStruct(fieldVal.Interface(), dict)
		} else if fieldVal.Kind() == reflect.Slice {
			if fieldVal.Len() > 0 {
				elem := fieldVal.Index(0)
				if elem.Kind() == reflect.Ptr {
					elem = elem.Elem()
				}
				if elem.Kind() == reflect.Struct {
					extractDictFromStruct(elem.Interface(), dict)
				}
			}
		}
	}
}

// GetJSONDict returns a JSON-compatible dictionary
func GetJSONDict(toolName string, v interface{}) string {
	dict := make(map[string]string)
	extractDictFromStruct(v, dict)

	var keys []string
	for short := range dict {
		keys = append(keys, short)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("{")
	for idx, short := range keys {
		long := dict[short]
		if idx > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf(`"%s":"%s"`, short, long))
	}
	sb.WriteString("}")
	return sb.String()
}

// compactXML transforms XML by replacing long attribute names with short ones
// Uses registered dictionaries or falls back to struct tag extraction
// Set AICT_NOCOMPACT=1 to disable compaction (for tests)
func compactXML(xmlStr string, v interface{}, toolName string) string {
	if os.Getenv("AICT_NOCOMPACT") == "1" {
		return xmlStr
	}
	// Try to get registered dictionary first (short -> long)
	shortToLong := GetRegisteredDict(toolName)
	if shortToLong == nil {
		shortToLong = make(map[string]string)
		extractAttrMapping(v, shortToLong)
	}

	// Invert to get long -> short for replacement
	longToShort := make(map[string]string)
	for short, long := range shortToLong {
		longToShort[long] = short
	}

	// Match: space+attr, >+attr, or tagname+space+attr (first attr after tag name)
	pattern := regexp.MustCompile(`([ >]|<[a-z]+ )([a-z_]+)="`)
	
	result := pattern.ReplaceAllStringFunc(xmlStr, func(match string) string {
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

	// Convert boolean strings to 1/0
	result = regexp.MustCompile(`="true"`).ReplaceAllString(result, `="1"`)
	result = regexp.MustCompile(`="false"`).ReplaceAllString(result, `="0"`)

	return result
}

// extractAttrMapping extracts attribute name mappings from struct tags
func extractAttrMapping(v interface{}, attrs map[string]string) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		xmlTag := field.Tag.Get("xml")
		if xmlTag == "" || xmlTag == "-" {
			continue
		}
		xmlParts := strings.Split(xmlTag, ",")
		xmlName := xmlParts[0]
		if xmlName == "" {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		jsonParts := strings.Split(jsonTag, ",")
		shortName := jsonParts[0]
		if shortName == "" {
			continue
		}

		attrs[xmlName] = shortName

		// Recurse into nested structs
		if fieldVal.Kind() == reflect.Ptr {
			fieldVal = fieldVal.Elem()
		}
		if fieldVal.Kind() == reflect.Struct {
			extractAttrMapping(fieldVal.Interface(), attrs)
		} else if fieldVal.Kind() == reflect.Slice {
			if fieldVal.Len() > 0 {
				elem := fieldVal.Index(0)
				if elem.Kind() == reflect.Ptr {
					elem = elem.Elem()
				}
				if elem.Kind() == reflect.Struct {
					extractAttrMapping(elem.Interface(), attrs)
				}
			}
		}
	}
}

// compactJSONKeys transforms JSON by replacing keys with short names
func compactJSONKeys(jsonStr string, v interface{}, toolName string) string {
	// Try to get registered dictionary first
	dict := GetRegisteredDict(toolName)
	if dict == nil {
		dict = make(map[string]string)
		extractDictFromStruct(v, dict)
	}

	// Build mapping: Go field name -> short name
	// dict is short -> long, we need to find Go field names
	fieldToShort := make(map[string]string)
	for short, long := range dict {
		// Find Go field name for this XML name
		goField := findGoFieldName(v, long)
		if goField != "" {
			fieldToShort[goField] = short
		}
	}

	result := jsonStr
	for goField, short := range fieldToShort {
		// Replace "GoField": with "short":
		pattern := regexp.MustCompile(`"` + regexp.QuoteMeta(goField) + `":`)
		result = pattern.ReplaceAllString(result, `"`+short+`":`)
	}

	// Convert boolean values
	result = regexp.MustCompile(`:true`).ReplaceAllString(result, `:1`)
	result = regexp.MustCompile(`:false`).ReplaceAllString(result, `:0`)

	return result
}

// findGoFieldName finds the Go struct field name for an XML attribute name
func findGoFieldName(v interface{}, xmlName string) string {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return ""
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		xmlTag := field.Tag.Get("xml")
		if xmlTag == "" || xmlTag == "-" {
			continue
		}
		xmlParts := strings.Split(xmlTag, ",")
		if xmlParts[0] == xmlName {
			return field.Name
		}

		// Recurse into nested structs
		if fieldVal.Kind() == reflect.Ptr {
			fieldVal = fieldVal.Elem()
		}
		if fieldVal.Kind() == reflect.Struct {
			if result := findGoFieldName(fieldVal.Interface(), xmlName); result != "" {
				return result
			}
		} else if fieldVal.Kind() == reflect.Slice {
			if fieldVal.Len() > 0 {
				elem := fieldVal.Index(0)
				if elem.Kind() == reflect.Ptr {
					elem = elem.Elem()
				}
				if elem.Kind() == reflect.Struct {
					if result := findGoFieldName(elem.Interface(), xmlName); result != "" {
						return result
					}
				}
			}
		}
	}
	return ""
}

// extractFieldMapping extracts Go field name -> short name mapping
func extractFieldMapping(v interface{}, fieldToShort map[string]string) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		jsonParts := strings.Split(jsonTag, ",")
		shortName := jsonParts[0]
		if shortName == "" {
			continue
		}

		fieldToShort[field.Name] = shortName

		// Recurse into nested structs
		if fieldVal.Kind() == reflect.Ptr {
			fieldVal = fieldVal.Elem()
		}
		if fieldVal.Kind() == reflect.Struct {
			extractFieldMapping(fieldVal.Interface(), fieldToShort)
		} else if fieldVal.Kind() == reflect.Slice {
			if fieldVal.Len() > 0 {
				elem := fieldVal.Index(0)
				if elem.Kind() == reflect.Ptr {
					elem = elem.Elem()
				}
				if elem.Kind() == reflect.Struct {
					extractFieldMapping(elem.Interface(), fieldToShort)
				}
			}
		}
	}
}
