package aegis

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Route struct {
	Prefix string
	Target *url.URL
}

type Config struct {
	ListenAddr      string
	Routes          []Route
	RequestsPerMin  int
	Burst           int
	ShutdownTimeout time.Duration
	LogLevel        slog.Level
}

func LoadConfig() (Config, error) {
	cfg := Config{ListenAddr: value("AEGIS_LISTEN_ADDR", ":8080"), RequestsPerMin: intValue("AEGIS_RATE_LIMIT_RPM", 120), Burst: intValue("AEGIS_RATE_LIMIT_BURST", 30), ShutdownTimeout: durationValue("AEGIS_SHUTDOWN_TIMEOUT", 10*time.Second), LogLevel: logLevel(value("AEGIS_LOG_LEVEL", "info"))}
	if cfg.RequestsPerMin < 1 || cfg.Burst < 1 { return Config{}, fmt.Errorf("rate limits must be positive") }
	routes, err := parseRoutes(value("AEGIS_ROUTES", "/=http://localhost:9000"))
	if err != nil { return Config{}, err }
	cfg.Routes = routes
	return cfg, nil
}

func parseRoutes(raw string) ([]Route, error) {
	var routes []Route
	for _, item := range strings.Split(raw, ",") {
		parts := strings.SplitN(strings.TrimSpace(item), "=", 2)
		if len(parts) != 2 || !strings.HasPrefix(parts[0], "/") { return nil, fmt.Errorf("invalid route %q; expected /prefix=http://upstream", item) }
		target, err := url.ParseRequestURI(parts[1])
		if err != nil || target.Scheme == "" || target.Host == "" { return nil, fmt.Errorf("invalid upstream in route %q", item) }
		routes = append(routes, Route{Prefix: strings.TrimRight(parts[0], "/"), Target: target})
	}
	if len(routes) == 0 { return nil, fmt.Errorf("at least one route is required") }
	return routes, nil
}

func value(key, fallback string) string { if v := os.Getenv(key); v != "" { return v }; return fallback }
func intValue(key string, fallback int) int { v, err := strconv.Atoi(value(key, strconv.Itoa(fallback))); if err != nil { return fallback }; return v }
func durationValue(key string, fallback time.Duration) time.Duration { v, err := time.ParseDuration(value(key, fallback.String())); if err != nil { return fallback }; return v }
func logLevel(raw string) slog.Level { var level slog.Level; if err := level.UnmarshalText([]byte(raw)); err != nil { return slog.LevelInfo }; return level }
