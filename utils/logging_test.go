package utils

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNewJSONLogger_OneJSONEventPerLine(t *testing.T) {
	var buf bytes.Buffer
	NewJSONLogger(&buf).Info("server started", "port", "8081")
	var e map[string]any
	if err := json.Unmarshal(buf.Bytes(), &e); err != nil {
		t.Fatalf("not JSON: %v: %s", err, buf.String())
	}
	if e["msg"] != "server started" || e["level"] != "INFO" || e["port"] != "8081" || e["time"] == nil {
		t.Errorf("unexpected event: %v", e)
	}
}
