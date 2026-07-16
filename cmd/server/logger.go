package main

import (
	"encoding/json"
	"io"
	"os"
	"time"
)

// logger output can be overridden for tests.
var loggerOut io.Writer = os.Stderr

// LogEntry is the structure emitted as JSON.
type LogEntry struct {
	Timestamp string                 `json:"ts"`
	Resolver  string                 `json:"resolver,omitempty"`
	Level     string                 `json:"level"`
	Message   string                 `json:"msg,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// SetLoggerOutput allows tests to capture logger output.
func SetLoggerOutput(w io.Writer) {
	if w == nil {
		loggerOut = os.Stderr
		return
	}
	loggerOut = w
}

// Log writes a structured JSON entry to the configured output.
func Log(level, resolver, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Resolver:  resolver,
		Level:     level,
		Message:   message,
		Fields:    fields,
	}
	b, _ := json.Marshal(entry)
	b = append(b, '\n')
	_, _ = loggerOut.Write(b)
}
