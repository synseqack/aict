package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/synseqack/aict/internal/tool"

	_ "github.com/synseqack/aict/tools/awk"
	_ "github.com/synseqack/aict/tools/basename"
	_ "github.com/synseqack/aict/tools/cat"
	_ "github.com/synseqack/aict/tools/checksums"
	_ "github.com/synseqack/aict/tools/completions"
	_ "github.com/synseqack/aict/tools/cut"
	_ "github.com/synseqack/aict/tools/df"
	_ "github.com/synseqack/aict/tools/diff"
	_ "github.com/synseqack/aict/tools/dirname"
	_ "github.com/synseqack/aict/tools/doctor"
	_ "github.com/synseqack/aict/tools/du"
	_ "github.com/synseqack/aict/tools/env"
	_ "github.com/synseqack/aict/tools/file"
	_ "github.com/synseqack/aict/tools/find"
	_ "github.com/synseqack/aict/tools/git"
	_ "github.com/synseqack/aict/tools/grep"
	_ "github.com/synseqack/aict/tools/head"
	_ "github.com/synseqack/aict/tools/jq"
	_ "github.com/synseqack/aict/tools/ls"
	_ "github.com/synseqack/aict/tools/ps"
	_ "github.com/synseqack/aict/tools/pwd"
	_ "github.com/synseqack/aict/tools/realpath"
	_ "github.com/synseqack/aict/tools/sed"
	_ "github.com/synseqack/aict/tools/sort"
	_ "github.com/synseqack/aict/tools/stat"
	_ "github.com/synseqack/aict/tools/system"
	_ "github.com/synseqack/aict/tools/tail"
	_ "github.com/synseqack/aict/tools/tar"
	_ "github.com/synseqack/aict/tools/tr"
	_ "github.com/synseqack/aict/tools/uniq"
	_ "github.com/synseqack/aict/tools/wc"
)

func TestGenerateSchemaDirectly(t *testing.T) {
	type TestConfig struct {
		Path string `flag:"" desc:"The path to list"`
		All  bool   `flag:"" desc:"Show hidden files"`
		Help bool   `flag:""`
	}

	meta := tool.GenerateSchema("test", "Test tool", TestConfig{})
	t.Logf("Generated schema: %+v", meta)
	t.Logf("InputSchema: %+v", meta.InputSchema)

	schemaJSON, _ := json.Marshal(meta.InputSchema)
	t.Logf("Schema JSON: %s", string(schemaJSON))

	t.Logf("Type: %v", meta.InputSchema["type"])

	if meta.InputSchema["type"] == nil {
		t.Error("schema type is nil!")
	}
}

func TestMCPServerDiscoversAllTools(t *testing.T) {
	ctx := context.Background()

	tools := tool.AllMeta()

	if len(tools) == 0 {
		t.Fatal("expected tools to be registered, got none")
	}

	t.Logf("Found %d registered tools:", len(tools))

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
			t.Logf("warning: failed to marshal schema for %s: %v", name, err)
			continue
		}

		var schemaMap map[string]interface{}
		if err := json.Unmarshal(schemaJSON, &schemaMap); err != nil {
			t.Logf("warning: failed to unmarshal schema for %s: %v", name, err)
			continue
		}

		server.AddTool(&mcp.Tool{
			Name:        name,
			Description: meta.Description,
			InputSchema: schemaMap,
		}, toolHandler(name))
	}

	t1, t2 := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("failed to connect server: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	defer clientSession.Close()

	listResult, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}

	if len(listResult.Tools) != len(tools) {
		t.Errorf("expected %d tools, got %d", len(tools), len(listResult.Tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range listResult.Tools {
		toolNames[tool.Name] = true
	}

	for name := range tools {
		if !toolNames[name] {
			t.Errorf("expected tool %s not found in MCP server", name)
		}
	}

	t.Logf("All %d tools registered successfully in MCP server\n", len(listResult.Tools))
}

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		args     map[string]interface{}
		check    func(result []string) bool
	}{
		{
			name:     "ls with all flags",
			toolName: "ls",
			args:     map[string]interface{}{"all": true, "sortTime": true, "reverse": true},
			check: func(result []string) bool {
				return contains(result, "-a") && contains(result, "-t") && contains(result, "-r")
			},
		},
		{
			name:     "ls with path",
			toolName: "ls",
			args:     map[string]interface{}{"path": ".", "all": true, "recursive": true},
			check: func(result []string) bool {
				return contains(result, "-a") && contains(result, "-R")
			},
		},
		{
			name:     "grep with pattern and flags",
			toolName: "grep",
			args:     map[string]interface{}{"pattern": "func", "recursive": true, "caseInsensitive": true},
			check: func(result []string) bool {
				return contains(result, "-r") && contains(result, "-i")
			},
		},
		{
			name:     "grep with include",
			toolName: "grep",
			args:     map[string]interface{}{"pattern": "test", "include": "*.go"},
			check: func(result []string) bool {
				return contains(result, "--include") && contains(result, "*.go")
			},
		},
		{
			name:     "wc with flags",
			toolName: "wc",
			args:     map[string]interface{}{"lines": true, "words": true, "bytes": true},
			check: func(result []string) bool {
				return contains(result, "-l") && contains(result, "-w") && contains(result, "-c")
			},
		},
		{
			name:     "cat with line numbers",
			toolName: "cat",
			args:     map[string]interface{}{"lineNumbers": true},
			check: func(result []string) bool {
				return contains(result, "-n")
			},
		},
		{
			name:     "find with options",
			toolName: "find",
			args:     map[string]interface{}{"name": "*.go", "type": "f"},
			check: func(result []string) bool {
				return contains(result, "-name") && contains(result, "*.go") && contains(result, "-type") && contains(result, "f")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildArgs(tt.toolName, tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.check(result) {
				t.Errorf("check failed for result: %v", result)
			}
		})
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func TestToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"string true", "true", true},
		{"string 1", "1", true},
		{"string false", "false", false},
		{"string 0", "0", false},
		{"string empty", "", false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toBool(tt.input)
			if result != tt.expected {
				t.Errorf("toBool(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected map[string]interface{}
	}{
		{
			name:     "nil",
			input:    nil,
			expected: map[string]interface{}{},
		},
		{
			name:     "map",
			input:    map[string]interface{}{"path": "/tmp"},
			expected: map[string]interface{}{"path": "/tmp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseArgs(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseArgs(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFindAICTBinary(t *testing.T) {
	binary := findAICTBinary()

	if _, err := os.Stat(binary); err != nil {
		t.Skipf("aict not found at %s", binary)
	}

	t.Logf("found aict at: %s", binary)
}

func TestRunAICTIntegration(t *testing.T) {
	binaryPath := findAICTBinary()
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skipf("aict binary not found at %s, skipping integration test", binaryPath)
	}

	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result, err := runAICT([]string{"ls", dir})
	if err != nil {
		t.Fatalf("runAICT failed: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if output["ls"] == nil {
		t.Errorf("expected ls key in output, got %v", output)
	}
}
