package traefik_cloudflare_geoblock_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	geoblock "github.com/moonlight8978/traefik-cloudflare-geoblock"
)

func TestDefaultConfig(t *testing.T) {
	config := geoblock.CreateConfig()
	assertEqual(t, config.Mode, "include")
	assertLen(t, config.Countries, 0)
	assertEqual(t, config.AllowEmpty, true)
}

func TestIncludeMode(t *testing.T) {
	tests := []struct {
		name       string
		allowEmpty bool
		response   int
		country    string
	}{
		{
			name:       "country is in included list",
			allowEmpty: false,
			response:   http.StatusOK,
			country:    "US",
		},
		{
			name:       "country is not in included list",
			allowEmpty: false,
			response:   http.StatusForbidden,
			country:    "CN",
		},
		{
			name:       "country is empty",
			allowEmpty: false,
			response:   http.StatusForbidden,
			country:    "",
		},
		{
			name:       "country is empty and allow empty is true",
			allowEmpty: true,
			response:   http.StatusOK,
			country:    "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := geoblock.CreateConfig()
			cfg.Mode = "include"
			cfg.Countries = []string{"US", "VN"}
			cfg.AllowEmpty = test.allowEmpty

			handler, rw, req := setupReq(t, cfg)
			req.Header.Add("CF-IPCountry", test.country)
			handler.ServeHTTP(rw, req)

			assertEqual(t, rw.Code, test.response)
		})
	}
}

func TestExcludeMode(t *testing.T) {
	tests := []struct {
		name       string
		allowEmpty bool
		response   int
		country    string
	}{
		{
			name:       "country is in excluded list",
			allowEmpty: false,
			response:   http.StatusForbidden,
			country:    "US",
		},
		{
			name:       "country is not in excluded list",
			allowEmpty: false,
			response:   http.StatusOK,
			country:    "CN",
		},
		{
			name:       "country is empty",
			allowEmpty: false,
			response:   http.StatusForbidden,
			country:    "",
		},
		{
			name:       "country is empty and allow empty is true",
			allowEmpty: true,
			response:   http.StatusOK,
			country:    "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := geoblock.CreateConfig()
			cfg.Mode = "exclude"
			cfg.Countries = []string{"US", "VN"}
			cfg.AllowEmpty = test.allowEmpty

			handler, rw, req := setupReq(t, cfg)
			req.Header.Add("CF-IPCountry", test.country)
			handler.ServeHTTP(rw, req)

			assertEqual(t, rw.Code, test.response)
		})
	}
}

func setupReq(t *testing.T, cfg *geoblock.Config) (http.Handler, *httptest.ResponseRecorder, *http.Request) {
	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := geoblock.New(ctx, next, cfg, "cloudflare-geoblock")
	if err != nil {
		t.Fatal(err)
	}

	rw := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)

	if err != nil {
		t.Fatal(err)
	}

	return handler, rw, req
}

func assertEqual(t *testing.T, actual, expected any) {
	if actual != expected {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func assertLen(t *testing.T, actual []string, expected int) {
	if len(actual) != expected {
		t.Errorf("expected %v, got %v", expected, len(actual))
	}
}
