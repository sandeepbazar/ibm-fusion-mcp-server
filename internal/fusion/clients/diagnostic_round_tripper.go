package clients

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"k8s.io/klog/v2"
)

const (
	// maxBodyReadSize is the maximum number of bytes read from a response body for logging.
	maxBodyReadSize = 16 * 1024
	// maxHexDumpSize is the maximum number of bytes shown in a hex dump for protobuf responses.
	maxHexDumpSize = 256
)

var (
	logBodyMode     string
	logBodyModeOnce sync.Once
)

// getLogBodyMode returns the diagnostic body logging mode from the FUSION_LOG_BODY
// environment variable. Valid values are "none" (default), "summary", and "full".
func getLogBodyMode() string {
	logBodyModeOnce.Do(func() {
		logBodyMode = strings.ToLower(strings.TrimSpace(os.Getenv("FUSION_LOG_BODY")))
		if logBodyMode == "" {
			logBodyMode = "none"
		}
	})
	return logBodyMode
}

// DiagnosticRoundTripper wraps an http.RoundTripper and logs request/response
// diagnostics at klog V(6). The logging detail is controlled by the
// FUSION_LOG_BODY environment variable (none, summary, full).
type DiagnosticRoundTripper struct {
	delegate http.RoundTripper
}

func (d *DiagnosticRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	mode := getLogBodyMode()
	if mode == "none" {
		return d.delegate.RoundTrip(req)
	}

	resp, err := d.delegate.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	d.logDiagnostics(req, resp, mode)
	return resp, nil
}

func (d *DiagnosticRoundTripper) logDiagnostics(req *http.Request, resp *http.Response, mode string) {
	contentType := resp.Header.Get("Content-Type")
	isProtobuf := strings.Contains(contentType, "protobuf")

	// Read body up to the cap
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyReadSize))
	if err != nil {
		klog.V(6).Infof("[diagnostic] %s %s -> %d (failed to read body: %v)", req.Method, req.URL, resp.StatusCode, err)
		return
	}
	// Read any remaining bytes to detect truncation, then restore the full body
	remaining, _ := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(io.MultiReader(bytes.NewReader(body), bytes.NewReader(remaining)))

	bodySize := int64(len(body)) + int64(len(remaining))

	switch mode {
	case "summary":
		d.logSummary(req, resp, contentType, isProtobuf, body, bodySize)
	case "full":
		d.logSummary(req, resp, contentType, isProtobuf, body, bodySize)
		d.logFullBody(isProtobuf, body)
	}
}

func (d *DiagnosticRoundTripper) logSummary(req *http.Request, resp *http.Response, contentType string, isProtobuf bool, body []byte, bodySize int64) {
	if isProtobuf {
		klog.V(6).Infof("[diagnostic] %s %s -> %d Content-Type=%s protobuf %d bytes",
			req.Method, req.URL, resp.StatusCode, contentType, bodySize)
		return
	}

	// Try to extract JSON metadata
	var obj map[string]interface{}
	if err := json.Unmarshal(body, &obj); err != nil {
		klog.V(6).Infof("[diagnostic] %s %s -> %d Content-Type=%s %d bytes (non-JSON or parse error)",
			req.Method, req.URL, resp.StatusCode, contentType, bodySize)
		return
	}

	kind, _ := obj["kind"].(string)
	apiVersion, _ := obj["apiVersion"].(string)
	resourceVersion := ""
	if metadata, ok := obj["metadata"].(map[string]interface{}); ok {
		resourceVersion, _ = metadata["resourceVersion"].(string)
	}

	itemCount := -1
	if items, ok := obj["items"].([]interface{}); ok {
		itemCount = len(items)
	}

	if itemCount >= 0 {
		klog.V(6).Infof("[diagnostic] %s %s -> %d Content-Type=%s kind=%s apiVersion=%s resourceVersion=%s items=%d %d bytes",
			req.Method, req.URL, resp.StatusCode, contentType, kind, apiVersion, resourceVersion, itemCount, bodySize)
	} else {
		klog.V(6).Infof("[diagnostic] %s %s -> %d Content-Type=%s kind=%s apiVersion=%s resourceVersion=%s %d bytes",
			req.Method, req.URL, resp.StatusCode, contentType, kind, apiVersion, resourceVersion, bodySize)
	}
}

func (d *DiagnosticRoundTripper) logFullBody(isProtobuf bool, body []byte) {
	if isProtobuf {
		dumpSize := len(body)
		if dumpSize > maxHexDumpSize {
			dumpSize = maxHexDumpSize
		}
		klog.V(6).Infof("[diagnostic] body (hex, first %d bytes):\n%s", dumpSize, hex.Dump(body[:dumpSize]))
		return
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, body, "", "  "); err != nil {
		klog.V(6).Infof("[diagnostic] body (raw, %d bytes):\n%s", len(body), truncateString(string(body), maxBodyReadSize))
		return
	}
	klog.V(6).Infof("[diagnostic] body (json, %d bytes):\n%s", len(body), truncateString(pretty.String(), maxBodyReadSize))
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + fmt.Sprintf("... (truncated, %d total bytes)", len(s))
}
