package api

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const BaseURL = "https://seats.aero/partnerapi"
const maxErrorBody = 64 << 10

type Request struct {
	Method string
	Path   string
	Query  url.Values
	Body   io.Reader
}

type Client struct {
	baseURL string
	key     string
	version string
	http    *http.Client
}

func NewClient(baseURL, key, version string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"), key: key, version: version,
		http: &http.Client{
			Timeout: 60 * time.Second,
			// Never forward the custom authorization header to a redirect target.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func (c *Client) Do(request Request, stdout io.Writer) error {
	endpoint := c.baseURL + request.Path
	if query := request.Query.Encode(); query != "" {
		endpoint += "?" + query
	}
	req, err := http.NewRequest(request.Method, endpoint, request.Body)
	if err != nil {
		return &Error{Code: "network_error", Message: "could not build the API request"}
	}
	req.Header.Set("Partner-Authorization", c.key)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "seatsaero/"+c.version)
	if request.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		// Transport errors can include URLs or header values; do not echo them.
		return &Error{Code: "network_error", Message: "could not reach the API (connection, TLS, or timeout failure)"}
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		// Read enough lookahead to redact a key crossing the output limit.
		body, _ := io.ReadAll(io.LimitReader(res.Body, int64(maxErrorBody+len(c.key))))
		upstream := string(body)
		if c.key != "" {
			upstream = strings.ReplaceAll(upstream, c.key, "[REDACTED]")
		}
		if len(upstream) > maxErrorBody {
			upstream = upstream[:maxErrorBody]
		}
		return responseError(res.StatusCode, strings.TrimSpace(upstream), res.Header.Get("x-ratelimit-reset"))
	}
	w := &lastByteWriter{writer: stdout}
	if _, err := io.Copy(w, res.Body); err != nil {
		return &Error{Code: "network_error", Message: "could not finish streaming the API response", Status: res.StatusCode}
	}
	if w.last != '\n' {
		if _, err := io.WriteString(stdout, "\n"); err != nil {
			return &Error{Code: "network_error", Message: "could not write the API response", Status: res.StatusCode}
		}
	}
	return nil
}

type lastByteWriter struct {
	writer io.Writer
	last   byte
}

func (w *lastByteWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if n > 0 {
		w.last = p[n-1]
	}
	return n, err
}
