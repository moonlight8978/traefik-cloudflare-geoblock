package traefik_cloudflare_geoblock

import (
	"context"
	"html/template"
	"net/http"
	"slices"
)

type Config struct {
	Mode       string   `json:"mode,omitempty"`
	Countries  []string `json:"countries,omitempty"`
	AllowEmpty bool     `json:"allowEmpty,omitempty"`
}

func CreateConfig() *Config {
	return &Config{
		Mode:       "include",
		Countries:  []string{},
		AllowEmpty: true,
	}
}

type GeoBlock struct {
	next     http.Handler
	name     string
	template *template.Template

	mode       string
	allowEmpty bool
	countries  []string
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &GeoBlock{
		next:       next,
		name:       name,
		template:   template.New("CloudflareGeoBlock").Delims("[[", "]]"),
		mode:       config.Mode,
		allowEmpty: config.AllowEmpty,
		countries:  config.Countries,
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
		http.Error(rw, "Access denied", http.StatusForbidden)
		return
	}

	if a.mode == "exclude" && slices.Contains(a.countries, ipCountry) {
		http.Error(rw, "Access denied", http.StatusForbidden)
		return
	}

	a.next.ServeHTTP(rw, req)
}
