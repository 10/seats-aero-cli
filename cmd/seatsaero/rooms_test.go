package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoomsRequests(t *testing.T) {
	cases := []struct {
		name, method, path, query string
		args                      []string
	}{
		{"search", "GET", "/search", "end_date=2026-11-27&location=Tokyo&start_date=2026-11-20&take=50", []string{"search", "--location", "Tokyo", "--start-date", "2026-11-20", "--end-date", "2026-11-27"}},
		{"search filters", "GET", "/search", "end_date=not-a-date&max_points=0&min_cpp=1.25&ne_lat=1.5&ne_lng=104.1&nights=3&skip=0&source=hyatt%2Chilton&start_date=2026-11-20&sw_lat=0&sw_lng=-1.25&take=10", []string{"search", "--start-date", "2026-11-20", "--end-date", "not-a-date", "--sw-lat", "0", "--sw-lng=-1.25", "--ne-lat", "1.5", "--ne-lng", "104.1", "--nights", "3", "--source", "hyatt,hilton", "--max-points", "0", "--min-cpp", "1.25", "--take", "10", "--skip", "0"}},
		{"availability defaults", "GET", "/availability", "source=hyatt&take=50", []string{"availability", "--source", "hyatt"}},
		{"availability filters", "GET", "/availability", "brand=park&category=7%2C8&country=Japan&end_date=2026-11-27&hotel_id=hotel%2Fone%2Chotel-two&include_hotel=false&max_points=30000&min_cpp=0&nights=2&order_by=lowest_cash&rate_type=cash&room_type=suite&skip=50&source=hyatt&start_date=2026-11-20&take=100", []string{"availability", "--source", "hyatt", "--hotel-id", "hotel/one,hotel-two", "--brand", "park", "--category", "7,8", "--country", "Japan", "--start-date", "2026-11-20", "--end-date", "2026-11-27", "--nights", "2", "--rate-type", "cash", "--room-type", "suite", "--max-points", "30000", "--min-cpp", "0", "--order-by", "lowest_cash", "--include-hotel=false", "--skip", "50", "--take", "100"}},
		{"availability hotel only", "GET", "/availability", "hotel_id=hotel&include_hotel=true&take=50", []string{"availability", "--hotel-id", "hotel", "--include-hotel"}},
		{"details", "GET", "/availability/id%2Fpart%3Fq%23frag", "", []string{"details", "id/part?q#frag"}},
		{"hotels defaults", "GET", "/hotels", "take=50", []string{"hotels"}},
		{"hotels filters", "GET", "/hotels", "brand=park&category=7%2C8&city=Tokyo&country=United+States%2CJapan&id=one%2Ctwo&internal_id=tyoph&search=Park+Hyatt&skip=1000&source=hyatt&state=&take=1000", []string{"hotels", "--id", "one,two", "--internal-id", "tyoph", "--source", "hyatt", "--country", "United States,Japan", "--state=", "--city", "Tokyo", "--brand", "park", "--category", "7,8", "--search", "Park Hyatt", "--take", "1000", "--skip", "1000"}},
		{"refresh", "POST", "/hotels/id%2Fpart%3Fq%23frag/refresh", "", []string{"refresh", "id/part?q#frag"}},
		{"alerts defaults", "GET", "/alerts", "take=50", []string{"alerts"}},
		{"alerts filters", "GET", "/alerts", "include_expired=true&skip=10&take=10", []string{"alerts", "--include-expired", "--take", "10", "--skip", "10"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isolateConfig(t)
			requests := 0
			body := "{\"data\": [], \"has_more\": false}"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				requestBody, err := io.ReadAll(r.Body)
				if err != nil || len(requestBody) != 0 || r.Header.Get("Content-Type") != "" {
					t.Error("Rooms requests must not send a body or Content-Type")
				}
				if r.Method != tc.method || r.URL.EscapedPath() != "/partnerapi"+tc.path || r.URL.RawQuery != tc.query {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				if r.Header.Get("Partner-Authorization") != testKey || r.Header.Get("Accept") != "application/json" || r.Header.Get("User-Agent") != "seatsaero/"+version() || r.Header.Get("Cookie") != "" {
					t.Error("incorrect Rooms headers")
				}
				if tc.method == "POST" {
					w.WriteHeader(http.StatusAccepted)
				}
				io.WriteString(w, body)
			}))
			defer server.Close()
			var stdout, stderr bytes.Buffer
			// A wrong service selection cannot accidentally pass on the same server.
			code := runWithBaseURLs(append([]string{"rooms"}, tc.args...), &stdout, &stderr, "http://127.0.0.1:0", server.URL+"/partnerapi")
			if code != 0 || stdout.String() != body+"\n" || stderr.Len() != 0 || requests != 1 {
				t.Fatalf("exit=%d stdout=%q stderr=%q requests=%d", code, stdout.String(), stderr.String(), requests)
			}
		})
	}
}
