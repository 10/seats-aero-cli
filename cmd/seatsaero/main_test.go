package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testKey = "pro_test_secret_key"

func isolateConfig(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("SEATSAERO_API_KEY", testKey)
}

type errorEnvelope struct {
	Error struct {
		Code, Message, Upstream string
		Status                  int
	} `json:"error"`
}

func checkError(t *testing.T, args []string, baseURL string, wantExit int, wantCode string, status int) errorEnvelope {
	t.Helper()
	code, out, errOut := invoke(args, baseURL)
	var envelope errorEnvelope
	if code != wantExit || out != "" || strings.Count(errOut, "\n") != 1 || json.Unmarshal([]byte(errOut), &envelope) != nil || envelope.Error.Code != wantCode || envelope.Error.Status != status || envelope.Error.Message == "" || strings.Contains(errOut, testKey) {
		t.Fatalf("code=%d stdout=%q stderr=%q; want %d %s HTTP %d", code, out, errOut, wantExit, wantCode, status)
	}
	return envelope
}

func TestHTTPFailures(t *testing.T) {
	for _, tc := range []struct {
		status, exit int
		code, body   string
	}{
		{400, 2, "bad_request", "{}"},
		{401, 3, "unauthorized", "bad_partner_key"},
		{403, 3, "unauthorized", `{"error":true,"code":"pro_key_required"}`},
		{404, 4, "not_found", ""},
		{429, 5, "quota_exceeded", "user_rate_limit_exceeded"},
		{500, 1, "upstream_error", "internal server error"},
		{502, 1, "upstream_error", "bad gateway"},
		{418, 1, "upstream_error", "teapot"},
		{302, 1, "upstream_error", "redirect"},
	} {
		t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
			isolateConfig(t)
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.Header().Set("x-ratelimit-reset", "75660")
				w.Header().Set("Location", "/redirect-target")
				w.WriteHeader(tc.status)
				io.WriteString(w, " \n"+tc.body+" \n")
			}))
			defer server.Close()
			envelope := checkError(t, []string{"alerts"}, server.URL, tc.exit, tc.code, tc.status)
			if envelope.Error.Upstream != tc.body || requests != 1 {
				t.Fatalf("upstream=%q requests=%d", envelope.Error.Upstream, requests)
			}
			if tc.status == 429 && !strings.Contains(envelope.Error.Message, "21h1m") {
				t.Fatal("missing reset duration")
			}
		})
	}
}

func TestErrorBodyLimitAndRedaction(t *testing.T) {
	isolateConfig(t)
	for _, body := range []string{
		strings.Repeat("x", 100_000),
		"rejected " + testKey,
		strings.Repeat("x", (64<<10)-5) + testKey + "tail",
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401); io.WriteString(w, body) }))
		envelope := checkError(t, []string{"alerts"}, server.URL, 3, "unauthorized", 401)
		server.Close()
		if len(envelope.Error.Upstream) > 64<<10 {
			t.Fatal("error body exceeds 64 KiB")
		}
		if body[0] == 'x' && len(envelope.Error.Upstream) != 64<<10 {
			t.Fatal("unexpected truncation length")
		}
		if strings.Contains(envelope.Error.Upstream, "pro_") {
			t.Fatal("key prefix leaked at truncation boundary")
		}
	}
}

func TestNetworkFailure(t *testing.T) {
	isolateConfig(t)
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()
	checkError(t, []string{"alerts"}, server.URL, 1, "network_error", 0)
}

func TestConfigFailuresAndOverrides(t *testing.T) {
	isolateConfig(t)
	t.Setenv("SEATSAERO_API_KEY", "")
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++; io.WriteString(w, "{}") }))
	defer server.Close()
	checkError(t, []string{"alerts"}, server.URL, 3, "missing_api_key", 0)
	path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "seatsaero", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{"", "{broken", `{"api_key":42}`, `{"api_key":"` + testKey + `"} trailing`} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		checkError(t, []string{"alerts"}, server.URL, 1, "config_error", 0)
	}
	if requests != 0 {
		t.Fatal("missing key or malformed config made a request")
	}
	t.Setenv("SEATSAERO_API_KEY", testKey)
	if code, _, errOut := invoke([]string{"alerts"}, server.URL); code != 0 || errOut != "" {
		t.Fatalf("env must bypass malformed file: %d %s", code, errOut)
	}
	t.Setenv("SEATSAERO_API_KEY", "")
	if code, _, errOut := invoke([]string{"--api-key", testKey, "alerts"}, server.URL); code != 0 || errOut != "" {
		t.Fatalf("flag must bypass malformed file: %d %s", code, errOut)
	}
	if err := os.WriteFile(path, []byte(`{"api_key":"`+testKey+`","future_field":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := invoke([]string{"alerts"}, server.URL); code != 0 || errOut != "" {
		t.Fatalf("unknown fields must be ignored: %d %s", code, errOut)
	}
}

func TestDefaultConfigPathAndAuthFailure(t *testing.T) {
	isolateConfig(t)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", t.TempDir())
	code, out, errOut := invoke([]string{"auth", testKey}, "http://invalid.invalid")
	var result map[string]string
	if code != 0 || errOut != "" || json.Unmarshal([]byte(out), &result) != nil || result["config"] != filepath.Join(os.Getenv("HOME"), ".config", "seatsaero", "config.json") {
		t.Fatalf("default config path: %d %q %q", code, out, errOut)
	}
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, []byte("block directory creation"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", blocked)
	checkError(t, []string{"auth", testKey}, "http://invalid.invalid", 1, "config_error", 0)
}

func TestUsageErrors(t *testing.T) {
	isolateConfig(t)
	t.Setenv("SEATSAERO_API_KEY", "")
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	for _, args := range [][]string{
		{}, {"search", "SFO"}, {"availability"}, {"routes"}, {"trips"}, {"refresh"}, {"auth"}, {"history"},
		{"history", "route"}, {"history", "route", "--kind", "cash", "--start-date", "2026-09-01", "--end-date", "2026-09-14"},
		{"destinations"}, {"destinations", "--origin-airport", "SFO", "--destination-airport", "NRT"},
		{"search", "SFO", "NRT", "--take", "many"}, {"alerts", "--unknown"},
		{"--api-key", testKey, "alerts", testKey}, {"auth", testKey, testKey},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			code, out, errOut := invoke(args, server.URL)
			if code != 2 || out != "" || !strings.Contains(errOut, "seatsaero:") || strings.Contains(errOut, `{"error":`) || strings.Contains(errOut, testKey) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, out, errOut)
			}
		})
	}
	if requests != 0 {
		t.Fatal("usage error sent a request")
	}
}

func TestExplicitKeyWithoutConfigLocation(t *testing.T) {
	isolateConfig(t)
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Partner-Authorization") != testKey {
			t.Error("wrong key")
		}
		io.WriteString(w, "{}")
	}))
	defer server.Close()
	if code, _, errOut := invoke([]string{"alerts"}, server.URL); code != 0 || errOut != "" {
		t.Fatalf("env key requires config location: %d %s", code, errOut)
	}
	t.Setenv("SEATSAERO_API_KEY", "")
	if code, _, errOut := invoke([]string{"--api-key", testKey, "alerts"}, server.URL); code != 0 || errOut != "" {
		t.Fatalf("flag key requires config location: %d %s", code, errOut)
	}
	if requests != 2 {
		t.Fatalf("expected two requests, got %d", requests)
	}
}

func TestInterruptedResponse(t *testing.T) {
	isolateConfig(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		io.WriteString(w, "partial")
	}))
	defer server.Close()
	code, out, errOut := invoke([]string{"alerts"}, server.URL)
	var envelope errorEnvelope
	if code != 1 || out != "partial" || strings.Count(errOut, "\n") != 1 || json.Unmarshal([]byte(errOut), &envelope) != nil || envelope.Error.Code != "network_error" || envelope.Error.Status != 200 || strings.Contains(errOut, testKey) {
		t.Fatalf("interrupted response: %d %q %q", code, out, errOut)
	}
}

func TestHelpAndVersion(t *testing.T) {
	isolateConfig(t)
	t.Setenv("SEATSAERO_API_KEY", "")
	for _, command := range []string{"", "search", "availability", "trips", "routes", "history", "destinations", "refresh", "alerts", "auth"} {
		args := []string{"--help"}
		if command != "" {
			args = []string{command, "--help"}
		}
		code, out, errOut := invoke(args, "http://invalid.invalid")
		if code != 0 || !strings.Contains(out, "Usage:") || errOut != "" {
			t.Fatalf("help %s: %d %q %q", command, code, out, errOut)
		}
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"--version"}, &out, &errOut); code != 0 || !strings.HasPrefix(out.String(), "seatsaero ") || strings.TrimSpace(strings.TrimPrefix(out.String(), "seatsaero ")) == "" || errOut.Len() != 0 {
		t.Fatalf("version: %d %q %q", code, out.String(), errOut.String())
	}
}

func TestSuccessPassthrough(t *testing.T) {
	isolateConfig(t)
	for _, body := range []string{`{"data": []}`, "{\"data\":[]}\n", "[1, 2, 3]\n\n", "", "not JSON\x00\xff", strings.Repeat("x", 100_000)} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201); io.WriteString(w, body) }))
		code, out, errOut := invoke([]string{"routes", "--source", "aeroplan"}, server.URL)
		server.Close()
		want := body
		if !strings.HasSuffix(want, "\n") {
			want += "\n"
		}
		if code != 0 || out != want || errOut != "" {
			t.Fatalf("passthrough failed for %d-byte body: code=%d output length=%d stderr=%q", len(body), code, len(out), errOut)
		}
	}
}

// The server waits for stdout to receive the first chunk before sending the last.
// A buffered implementation deadlocks here instead of satisfying the contract.
func TestSuccessStreamsBeforeResponseFinishes(t *testing.T) {
	isolateConfig(t)
	reader, writer := io.Pipe()
	firstChunk := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "first")
		w.(http.Flusher).Flush()
		select {
		case <-firstChunk:
		case <-time.After(3 * time.Second):
			t.Error("first chunk was buffered")
		}
		io.WriteString(w, "last")
	}))
	defer server.Close()
	go func() {
		prefix := make([]byte, 5)
		if _, err := io.ReadFull(reader, prefix); err != nil || string(prefix) != "first" {
			t.Error("did not receive first chunk")
		}
		close(firstChunk)
		io.Copy(io.Discard, reader)
		reader.Close()
	}()
	var errOut bytes.Buffer
	code := runWithBaseURL([]string{"alerts"}, writer, &errOut, server.URL)
	writer.Close()
	if code != 0 || errOut.Len() != 0 {
		t.Fatalf("streaming: %d %s", code, errOut.String())
	}
}

func TestAuthAndKeyResolution(t *testing.T) {
	isolateConfig(t)
	t.Setenv("SEATSAERO_API_KEY", "")
	path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "seatsaero", "config.json")
	var seenKeys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenKeys = append(seenKeys, r.Header.Get("Partner-Authorization"))
		io.WriteString(w, "{}")
	}))
	defer server.Close()
	for _, key := range []string{"file-first", "file-overwrite"} {
		code, out, errOut := invoke([]string{"auth", key}, server.URL)
		var result map[string]string
		if code != 0 || errOut != "" || json.Unmarshal([]byte(out), &result) != nil || result["config"] != path || strings.Contains(out, key) || len(seenKeys) != 0 {
			t.Fatalf("auth failed: code=%d stdout=%q stderr=%q", code, out, errOut)
		}
		for target, mode := range map[string]os.FileMode{path: 0600, filepath.Dir(path): 0700} {
			info, err := os.Stat(target)
			if err != nil || info.Mode().Perm() != mode {
				t.Fatalf("incorrect permissions on %s: %v %v", target, info, err)
			}
		}
		data, err := os.ReadFile(path)
		var saved map[string]string
		if err != nil || json.Unmarshal(data, &saved) != nil || saved["api_key"] != key || len(saved) != 1 {
			t.Fatal("config does not contain the saved key")
		}
		// auth must restrict permissions again before writing an existing file.
		if err := os.Chmod(path, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct{ env, flag, want string }{
		{"", "", "file-overwrite"},
		{"env-key", "", "env-key"},
		{"env-key", "flag-key", "flag-key"},
	} {
		t.Setenv("SEATSAERO_API_KEY", tc.env)
		args := []string{"alerts"}
		if tc.flag != "" {
			args = append(args, "--api-key", tc.flag)
		}
		code, out, errOut := invoke(args, server.URL)
		if code != 0 || out != "{}\n" || errOut != "" || seenKeys[len(seenKeys)-1] != tc.want {
			t.Fatalf("key resolution failed: %d %q %q", code, out, errOut)
		}
	}
	// Reading a hand-created file must not repair its permissions.
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0644 {
		t.Fatal("read changed file permissions")
	}
}

func TestCommandRequests(t *testing.T) {
	cases := []struct {
		name, method, path, query, body string
		args                            []string
	}{
		{"search flags", "GET", "/search", "cabins=Business%2Cfirst&carriers=AC%2CNH&cursor=1789181918&destination_airport=NRT&end_date=not-a-date&include_filtered=true&include_trips=true&min_cabin_pct=0&minify_trips=true&only_direct_flights=true&order_by=lowest_mileage&origin_airport=SFO&skip=0&sources=Aeroplan%2Cunited&start_date=2026-10-01&take=100", "", []string{"search", "sfo", "nrt", "--cabins", "Business,first", "--carriers", "ac,nh", "--cursor", "1789181918", "--end-date", "not-a-date", "--include-filtered", "--include-trips", "--min-cabin-pct", "0", "--minify-trips", "--only-direct-flights", "--order-by", "lowest_mileage", "--skip", "0", "--sources", "Aeroplan,united", "--start-date", "2026-10-01", "--take", "100"}},
		{"search empty and false", "GET", "/search", "cabins=&destination_airport=NRT&origin_airport=SFO&take=-1", "", []string{"search", "sfo", "nrt", "--cabins=", "--take=-1", "--include-trips=false"}},
		{"availability defaults", "GET", "/availability", "source=aeroplan&take=50", "", []string{"availability", "--source", "aeroplan"}},
		{"availability flags", "GET", "/availability", "cabin=First&cursor=1789181918&destination_region=Asia&end_date=2026-10-31&include_filtered=true&min_cabin_pct=75&origin_region=North+America&skip=10&source=Aeroplan&start_date=2026-10-01&take=10", "", []string{"availability", "--source", "Aeroplan", "--cabin", "First", "--cursor", "1789181918", "--destination-region", "Asia", "--end-date", "2026-10-31", "--include-filtered", "--min-cabin-pct", "75", "--origin-region", "North America", "--skip", "10", "--start-date", "2026-10-01", "--take", "10"}},
		{"trips defaults", "GET", "/trips/MixedID", "", "", []string{"trips", "MixedID"}},
		{"trips flags", "GET", "/trips/id%2Fpart%3Fq%23frag", "include_filtered=true&min_cabin_pct=0", "", []string{"trips", "id/part?q#frag", "--include-filtered", "--min-cabin-pct", "0"}},
		{"routes", "GET", "/routes", "source=NewProgram", "", []string{"routes", "--source", "NewProgram"}},
		{"destinations origin", "GET", "/destinations", "origin_airport=SFO", "", []string{"destinations", "--origin-airport", "sfo"}},
		{"destinations destination", "GET", "/destinations", "destination_airport=NRT", "", []string{"destinations", "--destination-airport", "nrt"}},
		{"refresh", "POST", "/refresh", "", `{"availability_ids":["MixedID","other/id"]}`, []string{"refresh", "MixedID", "other/id"}},
		{"alerts", "GET", "/alerts", "", "", []string{"alerts"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateConfig(t)
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if r.Method != tc.method || r.URL.EscapedPath() != "/partnerapi"+tc.path || r.URL.RawQuery != tc.query || string(body) != tc.body {
					t.Errorf("got %s %s body=%q; want %s %s?%s body=%q", r.Method, r.URL, body, tc.method, tc.path, tc.query, tc.body)
				}
				if r.Header.Get("Partner-Authorization") != testKey || r.Header.Get("Accept") != "application/json" || r.Header.Get("User-Agent") != "seatsaero/"+version() || strings.Contains(r.URL.String(), testKey) {
					t.Error("incorrect authentication or headers")
				}
				if tc.method == "POST" && r.Header.Get("Content-Type") != "application/json" {
					t.Error("missing JSON content type")
				}
				io.WriteString(w, "[]\n")
			}))
			defer server.Close()
			code, out, errOut := invoke(tc.args, server.URL+"/partnerapi")
			if code != 0 || out != "[]\n" || errOut != "" || requests != 1 {
				t.Fatalf("code=%d stdout=%q stderr=%q requests=%d", code, out, errOut, requests)
			}
		})
	}
}

func invoke(args []string, baseURL string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := runWithBaseURL(args, &stdout, &stderr, baseURL)
	return code, stdout.String(), stderr.String()
}

func TestSearchDefaults(t *testing.T) {
	isolateConfig(t)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != "GET" || r.URL.Path != "/partnerapi/search" || r.URL.RawQuery != "destination_airport=NRT%2CHND&origin_airport=SFO%2CLAX&take=50" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		if r.Header.Get("Partner-Authorization") != testKey || r.Header.Get("Accept") != "application/json" || r.Header.Get("User-Agent") != "seatsaero/"+version() {
			t.Error("incorrect request headers")
		}
		io.WriteString(w, "{\"data\": []}")
	}))
	defer server.Close()
	code, out, errOut := invoke([]string{"search", "sfo,lax", "nrt,hnd"}, server.URL+"/partnerapi")
	if code != 0 || out != "{\"data\": []}\n" || errOut != "" || requests != 1 {
		t.Fatalf("code=%d stdout=%q stderr=%q requests=%d", code, out, errOut, requests)
	}
}
