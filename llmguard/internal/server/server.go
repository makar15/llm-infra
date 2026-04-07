package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/rs/zerolog"
	"github.com/local/llmguard/internal/scanners"
)

// Server is the llmguard HTTP server.
type Server struct {
	upstream       *url.URL
	proxy          *httputil.ReverseProxy
	inputScanners  []scanners.Scanner
	outputScanners []scanners.Scanner
	log            zerolog.Logger
}

// New creates a Server. upstreamURL is the LiteLLM base URL.
func New(
	upstreamURL string,
	input []scanners.Scanner,
	output []scanners.Scanner,
	log zerolog.Logger,
) (*Server, error) {
	u, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream URL %q: %w", upstreamURL, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.Transport = &http.Transport{
		MaxIdleConns:       100,
		IdleConnTimeout:    90 * time.Second,
		DisableCompression: false,
	}

	return &Server{
		upstream:       u,
		proxy:          proxy,
		inputScanners:  input,
		outputScanners: output,
		log:            log,
	}, nil
}

// Handler returns the root http.Handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/", s.handleProxy)
	return mux
}

// ── /health ───────────────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// ── proxy ─────────────────────────────────────────────────────────────────────

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log := s.log.With().Str("method", r.Method).Str("path", r.URL.Path).Logger()

	// Only inspect POST (chat completion) requests with a body.
	if r.Method == http.MethodPost && len(s.inputScanners) > 0 {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error().Err(err).Msg("failed to read request body")
			http.Error(w, "failed to read request", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		inputText := extractInputText(body)
		if result := scanners.RunAll(s.inputScanners, inputText); result != nil {
			log.Warn().
				Str("scanner", result.Scanner).
				Str("reason", result.Reason).
				Msg("input blocked")
			writeBlocked(w, result)
			return
		}
	}

	// Intercept the upstream response to run output scanners.
	if len(s.outputScanners) > 0 {
		s.proxyWithOutputScan(w, r, log, start)
		return
	}

	r.URL.Host = s.upstream.Host
	r.URL.Scheme = s.upstream.Scheme
	r.Host = s.upstream.Host
	s.proxy.ServeHTTP(w, r)

	log.Info().Dur("latency", time.Since(start)).Msg("proxied")
}

// proxyWithOutputScan performs the upstream call, scans the response body,
// and either forwards or blocks the response.
func (s *Server) proxyWithOutputScan(
	w http.ResponseWriter, r *http.Request,
	log zerolog.Logger, start time.Time,
) {
	upstreamReq, err := http.NewRequestWithContext(r.Context(), r.Method,
		s.upstream.String()+r.URL.RequestURI(), r.Body)
	if err != nil {
		log.Error().Err(err).Msg("failed to build upstream request")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	upstreamReq.Header = r.Header.Clone()

	resp, err := http.DefaultClient.Do(upstreamReq)
	if err != nil {
		log.Error().Err(err).Msg("upstream request failed")
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("failed to read upstream response")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	outputText := extractOutputText(respBody)
	if result := scanners.RunAll(s.outputScanners, outputText); result != nil {
		log.Warn().
			Str("scanner", result.Scanner).
			Str("reason", result.Reason).
			Msg("output blocked")
		writeBlocked(w, result)
		return
	}

	// Forward original headers + body.
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)

	log.Info().Dur("latency", time.Since(start)).Msg("proxied")
}

// ── helpers ───────────────────────────────────────────────────────────────────

type openAIRequest struct {
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Text string `json:"text"`
	} `json:"choices"`
}

func extractInputText(body []byte) string {
	var req openAIRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return string(body)
	}
	var sb bytes.Buffer
	for _, m := range req.Messages {
		sb.WriteString(m.Content)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func extractOutputText(body []byte) string {
	var resp openAIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return string(body)
	}
	var sb bytes.Buffer
	for _, c := range resp.Choices {
		if c.Message.Content != "" {
			sb.WriteString(c.Message.Content)
		} else {
			sb.WriteString(c.Text)
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

type blockedResponse struct {
	Error   string `json:"error"`
	Scanner string `json:"scanner"`
	Reason  string `json:"reason"`
}

func writeBlocked(w http.ResponseWriter, r *scanners.ScanResult) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(blockedResponse{
		Error:   "request blocked by safety scanner",
		Scanner: r.Scanner,
		Reason:  r.Reason,
	})
}
