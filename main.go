package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Build-time variables (set via ldflags)
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"

	// Prometheus metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "epochcloud_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "epochcloud_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	appInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "epochcloud_app_info",
			Help: "Application information",
		},
		[]string{"version", "commit", "app"},
	)
)

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type VersionResponse struct {
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	BuildTime   string `json:"buildTime"`
	Hostname    string `json:"hostname"`
	Environment string `json:"environment"`
}

type PageData struct {
	Version      string
	Commit       string
	BuildTime    string
	Hostname     string
	Environment  string
	Timestamp    string
	RybbitSiteID string
	RybbitHost   string
}

// statusRecorder wraps http.ResponseWriter to capture the status code
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// metricsMiddleware wraps a handler to record metrics
func metricsMiddleware(path string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sr := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next(sr, r)
		duration := time.Since(start).Seconds()
		httpRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(sr.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	}
}

const pageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Image Updater Demo | {{.Environment}}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #2d1b69 0%, #11998e 100%);
            min-height: 100vh;
            color: #fff;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 2rem;
        }
        .container { max-width: 600px; width: 100%; text-align: center; }
        h1 {
            font-size: 2.5rem;
            margin-bottom: 1rem;
            background: linear-gradient(90deg, #00ff88, #00d9ff);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .subtitle { font-size: 1.1rem; opacity: 0.9; margin-bottom: 2rem; }
        .card {
            background: rgba(255, 255, 255, 0.1);
            border-radius: 16px;
            padding: 2rem;
            backdrop-filter: blur(10px);
        }
        .info { display: flex; justify-content: space-between; padding: 0.75rem 0; border-bottom: 1px solid rgba(255,255,255,0.1); }
        .info:last-child { border-bottom: none; }
        .label { opacity: 0.7; }
        .value { font-family: monospace; color: #00ff88; }
        .env-badge {
            display: inline-block;
            padding: 0.5rem 1.5rem;
            border-radius: 50px;
            font-weight: 600;
            text-transform: uppercase;
            margin-top: 1.5rem;
        }
        .env-dev { background: #ff6b6b; }
        .env-staging { background: #feca57; color: #1a1a2e; }
        .env-prod { background: #00d26a; }
    </style>
    {{if .RybbitSiteID}}<!-- First-party proxy: /_a/ is rewritten to /api/ by Traefik and forwarded to Rybbit backend (bypasses ad blockers) --><script src="/_a/script.js" data-site-id="{{.RybbitSiteID}}" defer></script>{{end}}
</head>
<body>
    <div class="container">
        <h1>🔄 Image Updater Demo</h1>
        <p class="subtitle">ArgoCD Image Updater automatic deployment demo</p>
        <div class="card">
            <div class="info"><span class="label">Version</span><span class="value">{{.Version}}</span></div>
            <div class="info"><span class="label">Commit</span><span class="value">{{.Commit}}</span></div>
            <div class="info"><span class="label">Build Time</span><span class="value">{{.BuildTime}}</span></div>
            <div class="info"><span class="label">Hostname</span><span class="value">{{.Hostname}}</span></div>
            <div class="info"><span class="label">Timestamp</span><span class="value">{{.Timestamp}}</span></div>
        </div>
        <span class="env-badge env-{{.Environment}}">{{.Environment}}</span>
    </div>
</body>
</html>`

func main() {
	hostname, _ := os.Hostname()
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "dev"
	}

	rybbitSiteID := os.Getenv("RYBBIT_SITE_ID")
	rybbitHost := os.Getenv("RYBBIT_HOST")

	// Set app info metric
	appInfo.WithLabelValues(Version, Commit, "test-image-updater").Set(1)

	tmpl := template.Must(template.New("page").Parse(pageTemplate))

	// Health endpoint
	http.HandleFunc("/health", metricsMiddleware("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthResponse{
			Status:    "healthy",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}))

	// Version endpoint
	http.HandleFunc("/version", metricsMiddleware("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(VersionResponse{
			Version:     Version,
			Commit:      Commit,
			BuildTime:   BuildTime,
			Hostname:    hostname,
			Environment: env,
		})
	}))

	// Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Main page
	http.HandleFunc("/", metricsMiddleware("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, PageData{
			Version:      Version,
			Commit:       Commit,
			BuildTime:    BuildTime,
			Hostname:     hostname,
			Environment:  env,
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
			RybbitSiteID: rybbitSiteID,
			RybbitHost:   rybbitHost,
		})
	}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting test-image-updater %s on port %s", Version, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
