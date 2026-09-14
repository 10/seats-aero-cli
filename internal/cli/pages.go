package cli

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/10/seats-aero-cli/internal/api"
)

func (p *PageFlags) run(ctx *Context, request api.Request) error {
	if !p.All {
		if p.MaxPages != nil {
			return &api.Error{Code: "bad_request", Message: "--max-pages requires --all"}
		}
		return ctx.Client.Do(request, ctx.Stdout)
	}
	maxPages := 10
	if p.MaxPages != nil {
		maxPages = *p.MaxPages
	}
	if maxPages < 1 || (p.Skip != nil && *p.Skip < 0) {
		return &api.Error{Code: "bad_request", Message: "--all requires a positive --max-pages and a nonnegative --skip"}
	}
	result := struct {
		Data     []json.RawMessage `json:"data"`
		Count    int               `json:"count"`
		HasMore  bool              `json:"hasMore"`
		Cursor   int64             `json:"cursor"`
		NextSkip int64             `json:"nextSkip"`
		Pages    int               `json:"pages"`
	}{Data: []json.RawMessage{}}
	if p.Skip != nil {
		result.NextSkip = *p.Skip
	}
	seen := make(map[string]bool)
	for result.Pages < maxPages {
		var body bytes.Buffer
		if err := ctx.Client.Do(request, &body); err != nil {
			return err
		}
		var page struct {
			Data    []json.RawMessage `json:"data"`
			HasMore *bool             `json:"hasMore"`
			Cursor  *int64            `json:"cursor"`
		}
		if err := json.Unmarshal(body.Bytes(), &page); err != nil || page.Data == nil || page.HasMore == nil || page.Cursor == nil {
			return &api.Error{Code: "invalid_response", Message: "expected an API page with data, hasMore and cursor"}
		}
		if *page.HasMore && len(page.Data) == 0 {
			return &api.Error{Code: "invalid_response", Message: "API returned an empty page with hasMore=true; pagination cannot advance"}
		}
		if result.Pages == 0 {
			result.Cursor = *page.Cursor
			if p.Cursor != nil {
				result.Cursor = *p.Cursor
			}
		}
		for _, row := range page.Data {
			var item struct{ ID string }
			if err := json.Unmarshal(row, &item); err != nil || item.ID == "" {
				return &api.Error{Code: "invalid_response", Message: "API availability row is missing a string ID"}
			}
			if !seen[item.ID] {
				seen[item.ID] = true
				result.Data = append(result.Data, row)
			}
		}
		result.Pages++
		// Skip counts upstream rows, including overlaps removed from the output.
		result.NextSkip += int64(len(page.Data))
		result.HasMore = *page.HasMore
		if !result.HasMore {
			break
		}
		request.Query.Set("skip", strconv.FormatInt(result.NextSkip, 10))
		request.Query.Set("cursor", strconv.FormatInt(result.Cursor, 10))
	}
	result.Count = len(result.Data)
	// Buffer until every requested page succeeds, so errors leave stdout empty.
	return json.NewEncoder(ctx.Stdout).Encode(result)
}
