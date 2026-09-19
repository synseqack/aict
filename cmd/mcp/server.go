package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/synseqack/aict/internal/tool"
	xmlout "github.com/synseqack/aict/internal/xml"
)

func toBool(v interface{}) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	if s, ok := v.(string); ok {
		return s == "true" || s == "1"
	}
	return false
}

func boolPointer(b bool) *bool {
	return &b
}

func getString(args map[string]interface{}, key, defaultVal string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return defaultVal
}

func getInt(args map[string]interface{}, key string, defaultVal int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	return defaultVal
}

// buildArgs turns an MCP argument object into the argv aict's own parseFlags
// expects. Flags come first, then positional arguments in the order
// positionalInputs declares. Both stages are deterministic: a JSON object
// carries no order, so iterating it directly made the argv — and therefore
// `diff a b` and `cat f1 f2` — come out shuffled on a fraction of calls.
func buildArgs(toolName string, args map[string]interface{}) ([]string, error) {
	var result []string

	// Keys arrive in whatever case the caller chose; the tables are all
	// lowercase, so normalise once and read through the copy from here on.
	normalized := make(map[string]interface{}, len(args))
	for key, value := range args {
		if key != "" {
			normalized[strings.ToLower(key)] = value
		}
	}

	// Flags, sorted by property name so repeated calls with the same input
	// produce identical argv. Order does not matter to any aict parser, but
	// it matters to anyone reading a log or a test.
	flagKeys := make([]string, 0, len(normalized))
	for key := range normalized {
		flagKeys = append(flagKeys, key)
	}
	sort.Strings(flagKeys)

	for _, lowerKey := range flagKeys {
		// A declared positional reaches the tool positionally below.
		if _, isPositional := positionalProperty(toolName, lowerKey); isPositional {
			continue
		}
		flag, hasFlag := flagFor(toolName, lowerKey)
		if !hasFlag || flag == "" {
			continue
		}

		switch v := normalized[lowerKey].(type) {
		case bool:
			if v {
				result = append(result, flag)
			}
		case string:
			if v != "" {
				result = append(result, flag, v)
			}
		case float64:
			// Emit the value even when it is zero: `head -n 0` is a
			// meaningful request, not an omitted one.
			result = append(result, flag, strconv.Itoa(int(v)))
		case []interface{}:
			// A repeated flag such as checksums -a md5 -a sha256 arrives
			// from JSON as an array and is expanded in order.
			for _, item := range v {
				if s, ok := item.(string); ok && s != "" {
					result = append(result, flag, s)
				}
			}
		}
	}

	// Positionals, in the order the tool's argv requires.
	for _, p := range positionalOrder(toolName) {
		value, ok := normalized[p.property]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case string:
			if v != "" {
				result = append(result, v)
			}
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok && s != "" {
					result = append(result, s)
				}
			}
		case float64:
			result = append(result, strconv.Itoa(int(v)))
		}
	}

	// Anything the caller sent that neither table knows about still reaches
	// the tool, sorted rather than map-ordered, so the argv stays
	// reproducible while remaining permissive.
	var leftover []string
	for _, lowerKey := range flagKeys {
		if _, hasFlag := flagFor(toolName, lowerKey); hasFlag {
			continue
		}
		if _, isPositional := positionalProperty(toolName, lowerKey); isPositional {
			continue
		}
		switch v := normalized[lowerKey].(type) {
		case string:
			if v != "" && lowerKey != "help" && lowerKey != "xml" && lowerKey != "json" && lowerKey != "plain" {
				leftover = append(leftover, v)
			}
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok && s != "" {
					leftover = append(leftover, s)
				}
			}
		}
	}
	result = append(result, leftover...)

	return result, nil
}

func findAICTBinary() string {
	if binaryPath := os.Getenv("AICT_BINARY"); binaryPath != "" {
		return binaryPath
	}

	execPath, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(execPath)
		candidate := filepath.Join(dir, "aict")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	dir, err := os.Getwd()
	if err == nil {
		candidate := filepath.Join(dir, "aict")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return "aict"
}

func runAICT(args []string) (string, error) {
	binaryPath := findAICTBinary()

	aictArgs := append([]string{args[0], "--json"}, args[1:]...)

	cmd := exec.Command(binaryPath, aictArgs...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(exitErr.Stderr), fmt.Errorf("aict error: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("failed to run aict: %w", err)
	}

	return string(output), nil
}

func parseArgs(args any) map[string]interface{} {
	argsMap := make(map[string]interface{})
	if args == nil {
		return argsMap
	}
	data, err := json.Marshal(args)
	if err == nil {
		_ = json.Unmarshal(data, &argsMap)
	}
	return argsMap
}

func toolHandler(toolName string) func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		argsMap := parseArgs(req.Params.Arguments)

		if toBool(argsMap["help"]) {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("usage: aict %s [options]", toolName)},
				},
			}, nil
		}

		aictArgs, err := buildArgs(toolName, argsMap)
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("error building args: %v", err)},
				},
			}, nil
		}

		output, err := runAICT(append([]string{toolName}, aictArgs...))
		if err != nil {
			return &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{Text: err.Error()},
				},
			}, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: annotateWithLegend(toolName, output)},
			},
		}, nil
	}
}

// annotateWithLegend adds a tool's short-to-long name map to a compact JSON
// response. aict's compact mode is what makes it cheap to send, but it leaves
// an agent reading {"p":"x","fl":"","cs":true} with no way to decode the
// abbreviations; the legend lets it do that without a second round trip.
// Responses that are not a compact JSON object are returned untouched.
//
// Only the abbreviations actually present in the response are included, so a
// short answer carries a short legend. The needle is the full quoted key plus
// the separating colon, which cannot match inside a longer key: in
// {"name":"x"} the sequence `"n":` does not occur, because the character
// after `"n` is `a`.
func annotateWithLegend(toolName, output string) string {
	dict := xmlout.GetRegisteredDict(toolName)
	if len(dict) == 0 {
		return output
	}
	if len(output) < 2 || output[0] != '{' {
		return output
	}

	keys := make([]string, 0, len(dict))
	for short := range dict {
		needle := `"` + short + `":`
		if strings.Contains(output, needle) {
			keys = append(keys, short)
		}
	}
	if len(keys) == 0 {
		return output
	}
	sort.Strings(keys)

	var b strings.Builder
	b.Grow(len(output) + len(keys)*24)
	b.WriteString(`{"_legend":{`)
	for i, short := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		legendEntry(&b, short)
		b.WriteByte(':')
		legendEntry(&b, dict[short])
	}
	b.WriteString(`}`)

	// A trailing comma before a bare closing brace would be invalid JSON, so
	// an object that carries no fields of its own absorbs the brace here.
	body := strings.TrimSpace(output[1:])
	if body != "}" {
		b.WriteByte(',')
	}
	b.WriteString(body)
	return b.String()
}

// legendEntry writes a string as a compact JSON literal.
func legendEntry(b *strings.Builder, s string) {
	enc := json.NewEncoder(b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	// Encode appends a newline; drop it so the surrounding object stays
	// on one line.
	str := b.String()
	b.Reset()
	b.WriteString(strings.TrimSuffix(str, "\n"))
}

func Serve() error {
	tools := tool.AllMeta()

	log.Printf("aict MCP server starting with %d tools...", len(tools))

	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "aict",
			Version: "1.0.0",
		},
		nil,
	)

	for name, meta := range tools {
		schemaJSON, err := json.Marshal(meta.InputSchema)
		if err != nil {
			log.Printf("warning: failed to marshal schema for %s: %v", name, err)
			continue
		}

		var schemaMap map[string]interface{}
		if err := json.Unmarshal(schemaJSON, &schemaMap); err != nil {
			log.Printf("warning: failed to unmarshal schema for %s: %v", name, err)
			continue
		}

		mergePositionalSchema(name, schemaMap)

		server.AddTool(&mcp.Tool{
			Name:        name,
			Description: meta.Description,
			InputSchema: schemaMap,
			Annotations: &mcp.ToolAnnotations{
				ReadOnlyHint:    true,
				DestructiveHint: boolPointer(false),
			},
		}, toolHandler(name))
	}

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("mcp server: %w", err)
	}
	return nil
}
