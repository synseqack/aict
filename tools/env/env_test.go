package env

import (
	"bytes"
	"encoding/xml"
	"os"
	"testing"
)

func runEnvWithResult(t *testing.T) *EnvResult {
	t.Helper()
	os.Setenv("AICT_XML", "1")
	defer os.Unsetenv("AICT_XML")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var outBuf bytes.Buffer
	done := make(chan struct{})
	go func() {
		outBuf.ReadFrom(r)
		close(done)
	}()

	Run([]string{})
	w.Close()
	os.Stdout = oldStdout
	<-done

	var result EnvResult
	if err := xml.Unmarshal(outBuf.Bytes(), &result); err != nil {
		t.Fatalf("invalid XML: %v", err)
	}
	return &result
}

func TestEnv_PathParsed(t *testing.T) {
	result := runEnvWithResult(t)

	if len(result.Path) == 0 {
		t.Error("expected PATH entries to be parsed")
	}

	for _, entry := range result.Path {
		if entry.Path == "" {
			t.Error("expected path entry to have non-empty path")
		}
	}
}

func TestEnv_SecretRedacted(t *testing.T) {
	os.Setenv("MY_API_KEY", "super_secret_value")
	defer os.Unsetenv("MY_API_KEY")

	result := runEnvWithResult(t)

	for _, v := range result.Variables {
		if v.Name == "MY_API_KEY" {
			if v.Redacted != "true" {
				t.Errorf("expected MY_API_KEY to be redacted, got Redacted=%q", v.Redacted)
			}
			return
		}
	}
	t.Error("MY_API_KEY not found in env result")
}

func TestEnv_TypeNumeric(t *testing.T) {
	os.Setenv("TEST_PORT_NUMBER", "8080")
	defer os.Unsetenv("TEST_PORT_NUMBER")

	result := runEnvWithResult(t)

	for _, v := range result.Variables {
		if v.Name == "TEST_PORT_NUMBER" {
			if v.Type != "numeric" {
				t.Errorf("expected type 'numeric' for PORT=8080, got %q", v.Type)
			}
			return
		}
	}
	t.Error("TEST_PORT_NUMBER not found in env result")
}

func TestEnv_TypeBool(t *testing.T) {
	os.Setenv("TEST_DEBUG_FLAG", "true")
	defer os.Unsetenv("TEST_DEBUG_FLAG")

	result := runEnvWithResult(t)

	for _, v := range result.Variables {
		if v.Name == "TEST_DEBUG_FLAG" {
			if v.Type != "boolean" {
				t.Errorf("expected type 'boolean' for value 'true', got %q", v.Type)
			}
			return
		}
	}
	t.Error("TEST_DEBUG_FLAG not found in env result")
}

func TestEnv_TypeURL(t *testing.T) {
	os.Setenv("TEST_DATABASE_URL", "postgres://localhost/mydb")
	defer os.Unsetenv("TEST_DATABASE_URL")

	result := runEnvWithResult(t)

	for _, v := range result.Variables {
		if v.Name == "TEST_DATABASE_URL" {
			if v.Type != "url" {
				t.Errorf("expected type 'url' for postgres URL, got %q", v.Type)
			}
			return
		}
	}
	t.Error("TEST_DATABASE_URL not found in env result")
}

func TestEnv_XMLValidity(t *testing.T) {
	result := runEnvWithResult(t)
	if result.XMLName.Local != "env" {
		t.Errorf("expected root element 'env', got %q", result.XMLName.Local)
	}
}
