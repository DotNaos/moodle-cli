package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type CommandRoute struct {
	APIPath     string
	CommandPath []string
	Summary     string
	Description string
	Stream      bool
}

type CommandRequest struct {
	Arguments []string `json:"arguments"`
}

type CommandRunner func(ctx context.Context, commandPath []string, arguments []string, stdout io.Writer, stderr io.Writer) error

func registerCommandRoutes(router interface {
	MethodFunc(string, string, http.HandlerFunc)
}, opts ServerOptions) {
	if opts.CommandRunner == nil {
		return
	}
	for _, route := range opts.CommandRoutes {
		router.MethodFunc(http.MethodPost, route.APIPath, commandHandler(opts.CommandRunner, route))
	}
}

func commandHandler(run CommandRunner, route CommandRoute) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeCommandRequest(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		if route.Stream {
			executeStreamingCommand(w, r, run, route, req)
			return
		}
		executeBufferedCommand(w, r, run, route, req)
	}
}

func decodeCommandRequest(body io.ReadCloser) (CommandRequest, error) {
	defer body.Close()

	var req CommandRequest
	if body == nil {
		return req, nil
	}
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return req, nil
		}
		return CommandRequest{}, fmt.Errorf("invalid command request body: %w", err)
	}
	return req, nil
}

func executeBufferedCommand(w http.ResponseWriter, r *http.Request, run CommandRunner, route CommandRoute, req CommandRequest) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run(r.Context(), route.CommandPath, req.Arguments, &stdout, &stderr)
	payload := bytes.TrimSpace(stdout.Bytes())
	if err == nil {
		if len(payload) == 0 {
			payload = []byte("{}")
		}
		writeRawJSON(w, http.StatusOK, payload)
		return
	}

	if len(payload) > 0 {
		writeRawJSON(w, commandHTTPStatus(payload), payload)
		return
	}

	message := strings.TrimSpace(stderr.String())
	if message == "" {
		message = err.Error()
	}
	writeError(w, http.StatusInternalServerError, errors.New(message))
}

func executeStreamingCommand(w http.ResponseWriter, r *http.Request, run CommandRunner, route CommandRoute, req CommandRequest) {
	flusher, _ := w.(http.Flusher)
	writer := &streamingResponseWriter{ResponseWriter: w, Flusher: flusher}
	var stderr bytes.Buffer

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	if flusher != nil {
		flusher.Flush()
	}

	err := run(r.Context(), route.CommandPath, req.Arguments, writer, &stderr)
	if err == nil {
		return
	}

	if writer.written > 0 {
		return
	}

	message := strings.TrimSpace(stderr.String())
	if message == "" {
		message = err.Error()
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": message})
}

type streamingResponseWriter struct {
	http.ResponseWriter
	Flusher http.Flusher
	written int
}

func (w *streamingResponseWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	w.written += n
	if err == nil && w.Flusher != nil {
		w.Flusher.Flush()
	}
	return n, err
}

func writeRawJSON(w http.ResponseWriter, status int, payload []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(payload)
	if len(payload) == 0 || payload[len(payload)-1] != '\n' {
		_, _ = w.Write([]byte("\n"))
	}
}

func commandHTTPStatus(payload []byte) int {
	var envelope struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return http.StatusInternalServerError
	}
	switch envelope.Code {
	case "invalid_arguments", "interactive_command", "output_format_conflict":
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
