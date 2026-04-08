package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/local/dashboard/internal/config"
)

// Server is the dashboard HTTP server.
type Server struct {
	cfg    *config.Config
	client *http.Client
	log    zerolog.Logger
}

func New(cfg *config.Config, log zerolog.Logger) *Server {
	return &Server{
		cfg: cfg,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		log: log,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/models", s.handleModels)
	mux.HandleFunc("/", s.handleUI)
	return mux
}

// ── /health ───────────────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// ── /api/status ───────────────────────────────────────────────────────────────

type serviceStatus struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Healthy bool   `json:"healthy"`
	Latency string `json:"latency_ms"`
}

type statusResponse struct {
	Timestamp string          `json:"timestamp"`
	Services  []serviceStatus `json:"services"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	checks := []struct {
		name string
		url  string
	}{
		{"litellm", s.cfg.Services.LiteLLM + "/health/liveliness"},
		{"langfuse", s.cfg.Services.LangFuse + "/api/public/health"},
		{"llmguard", s.cfg.Services.LLMGuard + "/health"},
	}

	results := make([]serviceStatus, len(checks))
	var wg sync.WaitGroup

	for i, c := range checks {
		wg.Add(1)
		go func(idx int, name, url string) {
			defer wg.Done()
			start := time.Now()
			resp, err := s.client.Get(url)
			latency := time.Since(start).Milliseconds()
			healthy := err == nil && resp != nil && resp.StatusCode < 400
			if resp != nil {
				resp.Body.Close()
			}
			results[idx] = serviceStatus{
				Name:    name,
				URL:     url,
				Healthy: healthy,
				Latency: fmt.Sprintf("%d", latency),
			}
		}(i, c.name, c.url)
	}
	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(statusResponse{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  results,
	})
}

// ── /api/models ───────────────────────────────────────────────────────────────

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, s.cfg.Services.LiteLLM+"/models", nil)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to build models request")
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if s.cfg.Services.LiteLLMAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.Services.LiteLLMAPIKey)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to fetch models from litellm")
		http.Error(w, `{"error":"litellm unavailable"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}

// ── / HTML UI ─────────────────────────────────────────────────────────────────

var uiTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>LLM Dashboard</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
           background: #0f1117; color: #e2e8f0; min-height: 100vh; padding: 2rem; }
    h1  { font-size: 1.5rem; font-weight: 600; margin-bottom: 2rem; color: #f8fafc; }
    h2  { font-size: 1rem; font-weight: 500; color: #94a3b8; margin-bottom: 1rem; }
    .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 1rem; margin-bottom: 2rem; }
    .card { background: #1e2230; border-radius: 8px; padding: 1.25rem; border: 1px solid #2d3748; }
    .card-title { font-size: 0.85rem; color: #94a3b8; text-transform: uppercase; letter-spacing: .05em; margin-bottom: .5rem; }
    .card-value { font-size: 1.4rem; font-weight: 600; }
    .dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 6px; }
    .healthy { color: #4ade80; } .dot.healthy { background: #4ade80; }
    .unhealthy { color: #f87171; } .dot.unhealthy { background: #f87171; }
    table { width: 100%; border-collapse: collapse; background: #1e2230; border-radius: 8px; overflow: hidden; }
    th, td { padding: .75rem 1rem; text-align: left; border-bottom: 1px solid #2d3748; font-size: .9rem; }
    th { color: #94a3b8; font-weight: 500; background: #161927; }
    tr:last-child td { border-bottom: none; }
    .tag { display: inline-block; padding: 2px 8px; border-radius: 4px;
           font-size: .75rem; background: #2d3748; color: #94a3b8; }
    #refresh { float: right; padding: .4rem .9rem; background: #3b82f6; color: #fff;
               border: none; border-radius: 6px; cursor: pointer; font-size: .85rem; }
    #refresh:hover { background: #2563eb; }
    #last-updated { font-size: .75rem; color: #64748b; margin-top: .5rem; }
  </style>
</head>
<body>
  <h1>LLM Dashboard <button id="refresh" onclick="loadAll()">Refresh</button></h1>

  <div class="grid" id="status-cards"></div>

  <h2>Available Models</h2>
  <table id="models-table">
    <thead><tr><th>Model ID</th><th>Provider</th></tr></thead>
    <tbody><tr><td colspan="2" style="color:#64748b">Loading...</td></tr></tbody>
  </table>
  <p id="last-updated"></p>

<script>
async function loadStatus() {
  const res = await fetch('/api/status');
  const data = await res.json();
  const cards = document.getElementById('status-cards');
  cards.innerHTML = data.services.map(s => ` + "`" + `
    <div class="card">
      <div class="card-title">${s.name}</div>
      <div class="card-value ${s.healthy ? 'healthy' : 'unhealthy'}">
        <span class="dot ${s.healthy ? 'healthy' : 'unhealthy'}"></span>
        ${s.healthy ? 'Healthy' : 'Unhealthy'}
      </div>
      <div style="margin-top:.5rem;font-size:.8rem;color:#64748b">${s.latency_ms}ms</div>
    </div>` + "`" + `).join('');
}

async function loadModels() {
  const tbody = document.querySelector('#models-table tbody');
  try {
    const res = await fetch('/api/models');
    const data = await res.json();
    const models = data.data || [];
    if (models.length === 0) {
      tbody.innerHTML = '<tr><td colspan="2" style="color:#64748b">No models found</td></tr>';
      return;
    }
    tbody.innerHTML = models.map(m => {
      const provider = m.id.split('/')[0] || 'unknown';
      return ` + "`" + `<tr><td>${m.id}</td><td><span class="tag">${provider}</span></td></tr>` + "`" + `;
    }).join('');
  } catch(e) {
    tbody.innerHTML = '<tr><td colspan="2" style="color:#f87171">Failed to load models</td></tr>';
  }
}

function loadAll() {
  loadStatus();
  loadModels();
  document.getElementById('last-updated').textContent =
    'Last updated: ' + new Date().toLocaleTimeString();
}

loadAll();
setInterval(loadAll, 30000);
</script>
</body>
</html>`

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, uiTemplate)
}
