package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	appInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_info",
			Help: "Application information",
		},
		[]string{"version", "pod_name", "node_name"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(appInfo)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

type PageData struct {
	PodName      string
	PodNamespace string
	PodIP        string
	NodeName     string
	Hostname     string
	Version      string
}

var pageTemplate = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>nine Webinar App</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
            color: #e2e8f0;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .container {
            max-width: 640px;
            width: 90%;
            padding: 2.5rem;
            background: rgba(30, 41, 59, 0.8);
            border-radius: 16px;
            border: 1px solid rgba(99, 102, 241, 0.3);
            box-shadow: 0 25px 50px rgba(0, 0, 0, 0.4);
        }
        h1 {
            font-size: 1.75rem;
            margin-bottom: 0.5rem;
            background: linear-gradient(90deg, #818cf8, #c084fc);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .subtitle {
            color: #94a3b8;
            margin-bottom: 2rem;
            font-size: 0.9rem;
        }
        .info-grid {
            display: grid;
            gap: 1rem;
        }
        .info-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 0.85rem 1rem;
            background: rgba(15, 23, 42, 0.6);
            border-radius: 8px;
            border: 1px solid rgba(51, 65, 85, 0.5);
        }
        .info-label {
            font-size: 0.8rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            color: #94a3b8;
        }
        .info-value {
            font-family: 'SF Mono', 'Fira Code', monospace;
            font-size: 0.85rem;
            color: #a5b4fc;
        }
        .version-badge {
            display: inline-block;
            margin-top: 1.5rem;
            padding: 0.3rem 0.8rem;
            background: rgba(99, 102, 241, 0.15);
            border: 1px solid rgba(99, 102, 241, 0.3);
            border-radius: 999px;
            font-size: 0.75rem;
            color: #818cf8;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>nine Webinar App</h1>
        <p class="subtitle">Running on NKE — Nine Kubernetes Engine</p>
        <div class="info-grid">
            <div class="info-item">
                <span class="info-label">Pod Name</span>
                <span class="info-value">{{.PodName}}</span>
            </div>
            <div class="info-item">
                <span class="info-label">Namespace</span>
                <span class="info-value">{{.PodNamespace}}</span>
            </div>
            <div class="info-item">
                <span class="info-label">Pod IP</span>
                <span class="info-value">{{.PodIP}}</span>
            </div>
            <div class="info-item">
                <span class="info-label">Node Name</span>
                <span class="info-value">{{.NodeName}}</span>
            </div>
            <div class="info-item">
                <span class="info-label">Hostname</span>
                <span class="info-value">{{.Hostname}}</span>
            </div>
        </div>
        <span class="version-badge">{{.Version}}</span>
    </div>
</body>
</html>`))

func main() {
	hostname, _ := os.Hostname()
	version := getEnv("APP_VERSION", "dev")

	data := PageData{
		PodName:      getEnv("POD_NAME", hostname),
		PodNamespace: getEnv("POD_NAMESPACE", "unknown"),
		PodIP:        getEnv("POD_IP", "unknown"),
		NodeName:     getEnv("NODE_NAME", "unknown"),
		Hostname:     hostname,
		Version:      version,
	}

	appInfo.WithLabelValues(version, data.PodName, data.NodeName).Set(1)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		start := time.Now()

		// Simulate slight variance in response time for interesting metrics
		time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := pageTemplate.Execute(w, data); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			httpRequestsTotal.WithLabelValues(r.Method, "/", "500").Inc()
			return
		}

		httpRequestsTotal.WithLabelValues(r.Method, "/", "200").Inc()
		httpRequestDuration.WithLabelValues(r.Method, "/").Observe(time.Since(start).Seconds())
	})

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})

	http.Handle("/metrics", promhttp.Handler())

	port := getEnv("PORT", "8080")
	log.Printf("Starting server on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
