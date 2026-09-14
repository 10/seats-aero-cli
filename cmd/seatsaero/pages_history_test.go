package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCollectedPages(t *testing.T) {
	for _, command := range [][]string{{"search", "SFO", "CAN", "--sources", "aeroplan"}, {"availability", "--source", "aeroplan"}} {
		for _, mode := range []string{"complete", "budget", "resume"} {
			t.Run(command[0]+"/"+mode, func(t *testing.T) {
				isolateConfig(t)
				requests := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests++
					q := r.URL.Query()
					if r.URL.Path != "/partnerapi/"+command[0] || q.Get("take") != "10" || r.Header.Get("Partner-Authorization") != testKey {
						t.Errorf("wrong request: %s", r.URL)
					}
					if q.Get("sources") != "aeroplan" && q.Get("source") != "aeroplan" {
						t.Error("program filter lost")
					}
					if q.Get("skip") != "" && q.Get("cursor") != "9007199254740993" {
						t.Error("original cursor lost or rounded")
					}
					switch q.Get("skip") {
					case "":
						io.WriteString(w, `{"data":[{"ID":"a","large":9007199254740993},{"ID":"b"}],"hasMore":true,"cursor":9007199254740993}`)
					case "2":
						io.WriteString(w, `{"data":[{"ID":"b"},{"ID":"c"}],"hasMore":true,"cursor":123}`)
					case "4":
						io.WriteString(w, `{"data":[{"ID":"d"}],"hasMore":false,"cursor":456}`)
					default:
						t.Errorf("skip must count duplicates: %s", q.Get("skip"))
						w.WriteHeader(400)
					}
				}))
				defer server.Close()
				args := append(append([]string{}, command...), "--all", "--take", "10")
				wantIDs, wantPages, wantSkip, wantMore := "abcd", 3, 5, false
				if mode == "budget" {
					args = append(args, "--max-pages", "2")
					wantIDs, wantPages, wantSkip, wantMore = "abc", 2, 4, true
				}
				if mode == "resume" {
					args = append(args, "--cursor", "9007199254740993", "--skip", "4")
					wantIDs, wantPages = "d", 1
				}
				code, out, errOut := invoke(args, server.URL+"/partnerapi")
				var result struct {
					Data                   []struct{ ID string }
					Count, Pages, NextSkip int
					Cursor                 int64
					HasMore                bool
				}
				if code != 0 || errOut != "" || json.Unmarshal([]byte(out), &result) != nil {
					t.Fatalf("code=%d out=%s err=%s", code, out, errOut)
				}
				var ids string
				for _, row := range result.Data {
					ids += row.ID
				}
				if ids != wantIDs || result.Count != len(wantIDs) || result.Pages != wantPages || requests != wantPages || result.NextSkip != wantSkip || result.HasMore != wantMore || result.Cursor != 9007199254740993 {
					t.Fatalf("incorrect collection: %s (%d requests)", out, requests)
				}
				if mode != "resume" && !strings.Contains(out, `"large":9007199254740993`) {
					t.Fatal("row field was lost or rounded")
				}
			})
		}
	}
}

func TestCollectedPagesStopOnFailure(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		code       string
		exit       int
	}{
		{"HTTP error", "quota", 429, "quota_exceeded", 5},
		{"HTML", "<html>login</html>", 200, "invalid_response", 1},
		{"missing metadata", `{"data":[]}`, 200, "invalid_response", 1},
		{"null data", `{"data":null,"hasMore":false,"cursor":1}`, 200, "invalid_response", 1},
		{"empty unfinished page", `{"data":[],"hasMore":true,"cursor":1}`, 200, "invalid_response", 1},
		{"missing ID", `{"data":[{}],"hasMore":false,"cursor":1}`, 200, "invalid_response", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateConfig(t)
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if requests == 1 {
					io.WriteString(w, `{"data":[{"ID":"a"}],"hasMore":true,"cursor":1}`)
					return
				}
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer server.Close()
			status := 0
			if tc.status != 200 {
				status = tc.status
			}
			checkError(t, []string{"search", "SFO", "CAN", "--all"}, server.URL, tc.exit, tc.code, status)
			if requests != 2 {
				t.Fatalf("retried failure: %d requests", requests)
			}
		})
	}
}

func TestPageBudgetAndValidation(t *testing.T) {
	isolateConfig(t)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		io.WriteString(w, `{"data":[{"ID":"a"}],"hasMore":true,"cursor":1}`)
	}))
	defer server.Close()
	for _, flags := range [][]string{{"--max-pages", "2"}, {"--all", "--max-pages", "0"}, {"--all", "--max-pages=-1"}, {"--all", "--skip=-1"}} {
		checkError(t, append([]string{"search", "SFO", "CAN"}, flags...), server.URL, 2, "bad_request", 0)
	}
	if requests != 0 {
		t.Fatal("invalid budget sent a request")
	}
	code, out, errOut := invoke([]string{"search", "SFO", "CAN", "--all"}, server.URL)
	if code != 0 || errOut != "" || requests != 10 || !strings.Contains(out, `"hasMore":true`) || !strings.Contains(out, `"nextSkip":10`) {
		t.Fatalf("default budget: %d %s %s requests=%d", code, out, errOut, requests)
	}
}

func TestHistory(t *testing.T) {
	for _, tc := range []struct {
		kind, endpoint, metric string
		extra                  []string
	}{
		{"availability", "daily_route_availability", "sum", nil},
		{"price", "daily_price_availability", "max", nil},
		{"price", "daily_price_availability", "min", []string{"--metric", "min", "--cabins", "j,f"}},
	} {
		t.Run(tc.kind+tc.metric, func(t *testing.T) {
			isolateConfig(t)
			body := `[{"ObservationDay":"2026-09-01T00:00:00Z","YMileageCost":55000}]`
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				q := r.URL.Query()
				if r.Method != "GET" || r.URL.EscapedPath() != "/_api/"+tc.endpoint+"/route%2Fpart%3Fx%23y" || q.Get("start") != "2026-09-01" || q.Get("end") != "2026-09-14" || q.Get("metric") != tc.metric || q.Get("stops") != "direct" {
					t.Errorf("wrong history request: %s %s", r.Method, r.URL)
				}
				if r.Header.Get("Partner-Authorization") != testKey || r.Header.Get("Cookie") != "" {
					t.Error("incorrect auth")
				}
				if tc.extra != nil && q.Get("cabins") != "J,F" {
					t.Error("cabin filter missing")
				}
				io.WriteString(w, body)
			}))
			defer server.Close()
			args := []string{"history", "route/part?x#y", "--kind", tc.kind, "--start-date", "2026-09-01", "--end-date", "2026-09-14"}
			code, out, errOut := invoke(append(args, tc.extra...), server.URL+"/partnerapi")
			if code != 0 || out != body+"\n" || errOut != "" || requests != 1 {
				t.Fatalf("%d %s %s requests=%d", code, out, errOut, requests)
			}
		})
	}
}

func TestHistoryRejectsUnexpectedResponses(t *testing.T) {
	for _, body := range []string{"<html>login</html>", "null", `[{}]`, `{"error":"login"}`, "redirect"} {
		t.Run(body, func(t *testing.T) {
			isolateConfig(t)
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if body == "redirect" {
					w.Header().Set("Location", "/login")
					w.WriteHeader(302)
				}
				io.WriteString(w, body)
			}))
			defer server.Close()
			status, errorCode := 0, "invalid_response"
			if body == "redirect" {
				status, errorCode = 302, "upstream_error"
			}
			checkError(t, []string{"history", "route", "--start-date", "2026-09-01", "--end-date", "2026-09-14"}, server.URL+"/partnerapi", 1, errorCode, status)
			if requests != 1 {
				t.Fatal("followed redirect")
			}
		})
	}
}
