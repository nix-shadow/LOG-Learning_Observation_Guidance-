// Package metrics (WP-4.3) records aggregate HTTP request statistics keyed
// by route PATTERN — never by user id, IP, or any personal data. The route
// pattern comes from gin's c.FullPath(), which yields the literal route
// ("/api/v1/users/:id/role"), so the counters are honest about shape and
// incapable of carrying PII into the public /metrics endpoint.
package metrics

import (
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// SpikeWindow is the sliding window for the 5xx alarm.
	SpikeWindow = 60 * time.Second
	// SpikeThreshold is the number of 5xx responses in SpikeWindow that
	// trips the alarm. Five server faults in a minute on a school LAN means
	// something is genuinely wrong — not one flaky request.
	SpikeThreshold = 5
)

// routeStats is the per-route counter set.
type routeStats struct {
	total    int
	byStatus map[int]int
}

// RouteSnapshot is the JSON-friendly view of one route's counters.
type RouteSnapshot struct {
	Method   string      `json:"method"`
	Route    string      `json:"route"`
	Total    int         `json:"total"`
	ByStatus map[int]int `json:"by_status"`
}

// Snapshot is the full aggregate view exposed to the admin endpoint.
type Snapshot struct {
	UptimeSeconds   int64           `json:"uptime_seconds"`
	TotalRequests   int             `json:"total_requests"`
	Total5xx        int             `json:"total_5xx"`
	SpikeActive     bool            `json:"spike_active"`
	LastAlert       *time.Time      `json:"last_alert_at,omitempty"`
	LastAlertDetail string          `json:"last_alert_detail,omitempty"`
	Routes          []RouteSnapshot `json:"routes"`
}

// Registry is a concurrency-safe counter store.
type Registry struct {
	mu       sync.Mutex
	started  time.Time
	routes   map[string]*routeStats // key: "METHOD routePattern"
	spikeAt  []time.Time            // recent 5xx timestamps (global, sliding window)
	alerted  bool                   // spike currently active (re-arms when the window empties)
	lastAt   *time.Time
	lastDesc string
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		started: time.Now(),
		routes:  make(map[string]*routeStats),
	}
}

// Record increments the counter for a route pattern and status. Patterns
// come from c.FullPath(); an empty pattern (unmatched route) is skipped so
// the counters never mix raw URLs into the route dimension.
func (r *Registry) Record(method, pattern string, status int) {
	if pattern == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	key := method + " " + pattern
	rs, ok := r.routes[key]
	if !ok {
		rs = &routeStats{byStatus: make(map[int]int)}
		r.routes[key] = rs
	}
	rs.total++
	rs.byStatus[status]++

	if status >= 500 {
		now := time.Now()
		r.spikeAt = append(r.spikeAt, now)
		cutoff := now.Add(-SpikeWindow)
		keep := 0
		for _, t := range r.spikeAt {
			if t.After(cutoff) {
				r.spikeAt[keep] = t
				keep++
			}
		}
		r.spikeAt = r.spikeAt[:keep]
		if len(r.spikeAt) >= SpikeThreshold && !r.alerted {
			r.alerted = true
			r.lastAt = &now
			r.lastDesc = formatSpike(r.spikeAt)
			slog.Warn("HTTP 5xx spike detected",
				"count_60s", len(r.spikeAt),
				"window_s", int(SpikeWindow.Seconds()),
			)
		}
	}
}

// Middleware records every request after the handler completes. It must be
// mounted inside the recovery middleware so panics (converted to 500s) are
// counted honestly.
func Middleware(r *Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		r.Record(c.Request.Method, c.FullPath(), c.Writer.Status())
	}
}

// Snapshot returns a copy of the aggregate counters.
func (r *Registry) Snapshot() Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Re-arm the alarm once the window has no qualifying 5xx events left.
	now := time.Now()
	cutoff := now.Add(-SpikeWindow)
	live := 0
	for _, t := range r.spikeAt {
		if t.After(cutoff) {
			live++
		}
	}
	if live == 0 {
		r.alerted = false
	}

	snap := Snapshot{
		UptimeSeconds:   int64(now.Sub(r.started).Seconds()),
		SpikeActive:     r.alerted,
		LastAlert:       r.lastAt,
		LastAlertDetail: r.lastDesc,
	}
	for key, rs := range r.routes {
		method, pattern, _ := strings.Cut(key, " ")
		snap.TotalRequests += rs.total
		for status, n := range rs.byStatus {
			if status >= 500 {
				snap.Total5xx += n
			}
		}
		byStatus := make(map[int]int, len(rs.byStatus))
		for s, n := range rs.byStatus {
			byStatus[s] = n
		}
		snap.Routes = append(snap.Routes, RouteSnapshot{
			Method:   method,
			Route:    pattern,
			Total:    rs.total,
			ByStatus: byStatus,
		})
	}
	sort.Slice(snap.Routes, func(i, j int) bool {
		if snap.Routes[i].Route == snap.Routes[j].Route {
			return snap.Routes[i].Method < snap.Routes[j].Method
		}
		return snap.Routes[i].Route < snap.Routes[j].Route
	})
	return snap
}

// RenderText emits a Prometheus-style text/plain view. Label values are
// route patterns only — no PII is ever rendered.
func (r *Registry) RenderText() string {
	snap := r.Snapshot()
	var b strings.Builder
	b.WriteString("# TYPE log_http_requests_total counter\n")
	for _, rs := range snap.Routes {
		for status, n := range rs.ByStatus {
			b.WriteString("log_http_requests_total{method=\"")
			b.WriteString(rs.Method)
			b.WriteString("\",route=\"")
			b.WriteString(rs.Route)
			b.WriteString("\",status=\"")
			b.WriteString(intStr(status))
			b.WriteString("\"} ")
			b.WriteString(intStr(n))
			b.WriteString("\n")
		}
	}
	b.WriteString("# TYPE log_http_requests counter\n")
	b.WriteString("log_http_requests ")
	b.WriteString(intStr(snap.TotalRequests))
	b.WriteString("\n")
	b.WriteString("# TYPE log_http_5xx counter\n")
	b.WriteString("log_http_5xx ")
	b.WriteString(intStr(snap.Total5xx))
	b.WriteString("\n")
	b.WriteString("# TYPE log_http_spike_active gauge\n")
	if snap.SpikeActive {
		b.WriteString("log_http_spike_active 1\n")
	} else {
		b.WriteString("log_http_spike_active 0\n")
	}
	return b.String()
}

func formatSpike(ts []time.Time) string {
	if len(ts) == 0 {
		return ""
	}
	start := ts[0]
	end := ts[len(ts)-1]
	return "5xx responses within " + end.Sub(start).Round(time.Second).String() + " window"
}

func intStr(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
