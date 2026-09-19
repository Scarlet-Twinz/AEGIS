package aegis

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync/atomic"
	"time"
)

type Metrics struct { requests, errors, limited, inFlight atomic.Uint64; latencyNanos atomic.Uint64 }
type Server struct { cfg Config; logger *slog.Logger; limiter *Limiter; metrics Metrics; proxies map[string]*httputil.ReverseProxy }

func NewServer(cfg Config, logger *slog.Logger) (*Server, error) {
	s := &Server{cfg:cfg, logger:logger, limiter:NewLimiter(cfg.RequestsPerMin,cfg.Burst), proxies:make(map[string]*httputil.ReverseProxy)}
	for _, route := range cfg.Routes {
		target := route.Target
		proxy := httputil.NewSingleHostReverseProxy(target)
		original := proxy.Director
		proxy.Director = func(r *http.Request) { original(r); r.Host = target.Host }
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) { s.metrics.errors.Add(1); s.logger.Error("upstream request failed", "request_id", r.Header.Get("X-Request-ID"), "error", err); http.Error(w, "upstream unavailable", http.StatusBadGateway) }
		s.proxies[route.Prefix] = proxy
	}
	return s, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux(); mux.HandleFunc("GET /health", s.health); mux.HandleFunc("GET /metrics", s.metricsHandler); mux.HandleFunc("/", s.proxy)
	return s.observe(mux)
}
func (s *Server) proxy(w http.ResponseWriter, r *http.Request) {
	if !s.limiter.Allow(clientIP(r), time.Now()) { s.metrics.limited.Add(1); http.Error(w,"rate limit exceeded",http.StatusTooManyRequests); return }
	prefix := s.routeFor(r.URL.Path); if prefix == "" { http.NotFound(w,r); return }; s.proxies[prefix].ServeHTTP(w,r)
}
func (s *Server) routeFor(path string) string { best := ""; for prefix := range s.proxies { if (prefix == "" || prefix == "/" || path == prefix || strings.HasPrefix(path,prefix+"/")) && len(prefix) > len(best) { best=prefix } }; return best }
func (s *Server) health(w http.ResponseWriter, _ *http.Request) { w.Header().Set("Content-Type","application/json"); _=json.NewEncoder(w).Encode(map[string]string{"status":"ok"}) }
func (s *Server) metricsHandler(w http.ResponseWriter, _ *http.Request) { requests:=s.metrics.requests.Load(); total:=s.metrics.latencyNanos.Load(); avg:=uint64(0); if requests>0 { avg=total/requests }; w.Header().Set("Content-Type","application/json"); _=json.NewEncoder(w).Encode(map[string]uint64{"requests_total":requests,"errors_total":s.metrics.errors.Load(),"rate_limited_total":s.metrics.limited.Load(),"in_flight":s.metrics.inFlight.Load(),"average_latency_ms":avg/uint64(time.Millisecond)}) }
func (s *Server) observe(next http.Handler) http.Handler { return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) { start:=time.Now(); id:=r.Header.Get("X-Request-ID"); if id=="" { id=newID(); r.Header.Set("X-Request-ID",id) }; w.Header().Set("X-Request-ID",id); s.metrics.requests.Add(1); s.metrics.inFlight.Add(1); defer s.metrics.inFlight.Add(^uint64(0)); next.ServeHTTP(w,r); elapsed:=time.Since(start); s.metrics.latencyNanos.Add(uint64(elapsed)); s.logger.Info("request complete", "request_id",id,"method",r.Method,"path",r.URL.Path,"remote_ip",clientIP(r),"latency_ms",elapsed.Milliseconds()) }) }
func clientIP(r *http.Request) string { host,_,err:=net.SplitHostPort(r.RemoteAddr); if err==nil { return host }; return r.RemoteAddr }
func newID() string { b:=make([]byte,12); if _,err:=rand.Read(b); err != nil { return time.Now().Format("20060102150405.000000000") }; return hex.EncodeToString(b) }
