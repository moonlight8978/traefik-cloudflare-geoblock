package traefik_cloudflare_geoblock

import (
	"context"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
)

type Config struct {
	Mode       string   `json:"mode,omitempty"`
	Countries  []string `json:"countries,omitempty"`
	AllowEmpty bool     `json:"allowEmpty,omitempty"`
	Exceptions []string `json:"exceptions,omitempty"`
}

func CreateConfig() *Config {
	return &Config{
		Mode:       "include",
		Countries:  []string{},
		AllowEmpty: true,
	}
}

type GeoBlock struct {
	next       http.Handler
	name       string
	template   *template.Template
	mode       string
	allowEmpty bool
	countries  []string
	exceptions []ipRange
}

type ipRange struct {
	start netip.Addr
	end   netip.Addr
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	exceptions, err := parseExceptions(config.Exceptions)
	if err != nil {
		return nil, err
	}

	return &GeoBlock{
		next:       next,
		name:       name,
		template:   template.New("CloudflareGeoBlock").Delims("[[", "]]"),
		mode:       config.Mode,
		allowEmpty: config.AllowEmpty,
		countries:  config.Countries,
		exceptions: exceptions,
	}, nil
}

func (a *GeoBlock) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	ipCountry := req.Header.Get("CF-IPCountry")

	if ipCountry == "" {
		if a.allowEmpty {
			a.next.ServeHTTP(rw, req)

			return
		}

		http.Error(rw, "Access denied", http.StatusForbidden)
		return
	}

	if a.mode == "include" && !slices.Contains(a.countries, ipCountry) {
		if a.isException(req) {
			a.next.ServeHTTP(rw, req)
			return
		}

		http.Error(rw, "Access denied", http.StatusForbidden)
		return
	}

	if a.mode == "exclude" && slices.Contains(a.countries, ipCountry) {
		http.Error(rw, "Access denied", http.StatusForbidden)
		return
	}

	a.next.ServeHTTP(rw, req)
}

func (a *GeoBlock) isException(req *http.Request) bool {
	if len(a.exceptions) == 0 {
		return false
	}

	ip, ok := getClientIP(req)
	if !ok {
		return false
	}

	for _, r := range a.exceptions {
		if ipInRange(ip, r) {
			return true
		}
	}

	return false
}

func getClientIP(req *http.Request) (netip.Addr, bool) {
	candidates := []string{
		req.Header.Get("CF-Connecting-IP"),
		req.Header.Get("X-Real-IP"),
	}

	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			candidates = append(candidates, strings.TrimSpace(parts[0]))
		}
	}

	if host, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		candidates = append(candidates, host)
	} else if req.RemoteAddr != "" {
		candidates = append(candidates, req.RemoteAddr)
	}

	for _, c := range candidates {
		if c == "" {
			continue
		}

		ip, err := netip.ParseAddr(c)
		if err == nil {
			return ip, true
		}
	}

	return netip.Addr{}, false
}

func ipInRange(ip netip.Addr, r ipRange) bool {
	if r.start.Compare(ip) > 0 {
		return false
	}

	if r.end.Compare(ip) < 0 {
		return false
	}

	return true
}

func parseExceptions(values []string) ([]ipRange, error) {
	if len(values) == 0 {
		return nil, nil
	}

	result := make([]ipRange, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if strings.Contains(value, "-") {
			start, end, err := parseIPRange(value)
			if err != nil {
				return nil, err
			}

			result = append(result, ipRange{start: start, end: end})
			continue
		}

		addr, err := netip.ParseAddr(value)
		if err != nil {
			return nil, err
		}

		result = append(result, ipRange{start: addr, end: addr})
	}

	return result, nil
}

func parseIPRange(value string) (netip.Addr, netip.Addr, error) {
	parts := strings.SplitN(value, "-", 2)
	if len(parts) != 2 {
		return netip.Addr{}, netip.Addr{}, fmt.Errorf("invalid IP range: %s", value)
	}

	start, err := netip.ParseAddr(strings.TrimSpace(parts[0]))
	if err != nil {
		return netip.Addr{}, netip.Addr{}, err
	}

	end, err := netip.ParseAddr(strings.TrimSpace(parts[1]))
	if err != nil {
		return netip.Addr{}, netip.Addr{}, err
	}

	if start.BitLen() != end.BitLen() {
		return netip.Addr{}, netip.Addr{}, fmt.Errorf("mismatched IP versions in range: %s", value)
	}

	if start.Compare(end) > 0 {
		return netip.Addr{}, netip.Addr{}, fmt.Errorf("invalid IP range (start > end): %s", value)
	}

	return start, end, nil
}
