package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	aipb "slate/services/ai-service/api/proto"
	"slate/services/ai-service/internal/cache"
	"slate/services/ai-service/internal/claude"
)

const cmdPaletteTTL = 30 * time.Second

const cmdPaletteSystemPrompt = `You are a navigation assistant for an LMS. ` +
	`Given a free-text user query and the user's role and enrolled courses, ` +
	`return up to 5 action suggestions as STRICT JSON: ` +
	`{"results":[{"label":"...","route":"/...","icon":"..."}]}. Only pick ` +
	`routes from the provided route_catalog — never invent a route. Icons ` +
	`must be a short kebab-case identifier (e.g. "book-open"). Return no ` +
	`text outside the JSON.`

type CmdPalette struct {
	claude    claude.Invoker
	cache     *cache.Cache
	model     string
	maxTokens int64
}

func NewCmdPalette(c claude.Invoker, cc *cache.Cache, model string, maxTokens int) *CmdPalette {
	return &CmdPalette{claude: c, cache: cc, model: model, maxTokens: int64(maxTokens)}
}

func (p *CmdPalette) Resolve(ctx context.Context, req *aipb.CmdPaletteRequest) (*aipb.CmdPaletteResponse, error) {
	if req == nil || strings.TrimSpace(req.Query) == "" {
		return nil, errors.New("cmd_palette: query required")
	}

	key := cache.CmdPaletteKey(req.TenantSlug, req.UserId, req.Query)
	var cached aipb.CmdPaletteResponse
	if hit, err := p.cache.GetJSON(ctx, key, &cached); err == nil && hit {
		cached.FromCache = true
		return &cached, nil
	}

	user := struct {
		Role    string                  `json:"role"`
		Courses []string                `json:"courses"`
		Query   string                  `json:"query"`
		Routes  []*aipb.CmdPaletteRoute  `json:"route_catalog"`
	}{req.Role, req.Courses, req.Query, req.RouteCatalog}
	body, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}
	result, err := p.claude.Invoke(ctx, req.TenantSlug, claude.Prompt{
		Model:        p.model,
		MaxTokens:    p.maxTokens,
		SystemPrompt: cmdPaletteSystemPrompt,
		UserText:     string(body),
		Purpose:      "cmd_palette",
	})
	if err != nil {
		return nil, err
	}

	results, err := parseCmdPalette(result.Text, req.RouteCatalog)
	if err != nil {
		return nil, err
	}
	resp := &aipb.CmdPaletteResponse{Results: results, FromCache: false}
	_ = p.cache.SetJSON(ctx, key, resp, cmdPaletteTTL)
	return resp, nil
}

// parseCmdPalette decodes Claude output and filters to only routes present
// in the catalog — this is the bounded-routes guarantee from W7.4.
func parseCmdPalette(raw string, catalog []*aipb.CmdPaletteRoute) ([]*aipb.CmdPaletteResult, error) {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "{"); i > 0 {
		raw = raw[i:]
	}
	if j := strings.LastIndex(raw, "}"); j >= 0 && j < len(raw)-1 {
		raw = raw[:j+1]
	}
	var wire struct {
		Results []struct {
			Label string `json:"label"`
			Route string `json:"route"`
			Icon  string `json:"icon"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(raw), &wire); err != nil {
		return nil, err
	}
	allowed := make(map[string]struct{}, len(catalog))
	for _, r := range catalog {
		allowed[r.Route] = struct{}{}
	}
	out := make([]*aipb.CmdPaletteResult, 0, len(wire.Results))
	for _, r := range wire.Results {
		if _, ok := allowed[r.Route]; !ok {
			continue // silently drop hallucinated routes
		}
		out = append(out, &aipb.CmdPaletteResult{Label: r.Label, Route: r.Route, Icon: r.Icon})
		if len(out) >= 5 {
			break
		}
	}
	return out, nil
}
