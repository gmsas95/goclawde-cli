// Package search provides web search capabilities using multiple providers
package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gmsas95/myrai-cli/internal/skills"
)

// SearchResult represents a single search result
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Snippet     string `json:"snippet"`
	PublishedAt string `json:"published_at,omitempty"`
	Source      string `json:"source"`
}

// SearchResponse represents the response from a search
type SearchResponse struct {
	Query        string         `json:"query"`
	Results      []SearchResult `json:"results"`
	TotalResults int            `json:"total_results"`
	Provider     string         `json:"provider"`
	SearchTime   time.Duration  `json:"search_time"`
}

// Provider interface for search providers
type Provider interface {
	Name() string
	Search(ctx context.Context, query string, numResults int) (*SearchResponse, error)
	IsAvailable() bool
}

// SearchSkill provides web search capabilities
type SearchSkill struct {
	*skills.BaseSkill
	config          Config
	providers       map[string]Provider
	defaultProvider string
}

// Config holds search configuration
type Config struct {
	Enabled     bool   `json:"enabled"`
	Provider    string `json:"provider"` // brave, serper, google, duckduckgo
	APIKey      string `json:"api_key"`
	MaxResults  int    `json:"max_results"`
	TimeoutSecs int    `json:"timeout_seconds"`
}

// NewSearchSkill creates a new search skill
func NewSearchSkill(cfg Config) *SearchSkill {
	if cfg.MaxResults == 0 {
		cfg.MaxResults = 5
	}
	if cfg.TimeoutSecs == 0 {
		cfg.TimeoutSecs = 30
	}

	s := &SearchSkill{
		BaseSkill: skills.NewBaseSkill("search", "Web search for real-time information from the internet", "1.0.0"),
		config:    cfg,
		providers: make(map[string]Provider),
	}

	s.registerProviders()
	s.registerTools()
	return s
}

// IsEnabled returns whether the skill is enabled
func (s *SearchSkill) IsEnabled() bool {
	return s.config.Enabled
}

func (s *SearchSkill) registerProviders() {
	// Register all providers
	s.providers["brave"] = NewBraveProvider(s.config.APIKey)
	s.providers["serper"] = NewSerperProvider(s.config.APIKey)
	s.providers["duckduckgo"] = NewDuckDuckGoProvider()
	s.providers["google"] = NewGoogleProvider(s.config.APIKey)

	// Set default provider
	if s.config.Provider != "" {
		s.defaultProvider = s.config.Provider
	} else {
		// Auto-select first available provider
		for name, provider := range s.providers {
			if provider.IsAvailable() {
				s.defaultProvider = name
				break
			}
		}
	}
}

func (s *SearchSkill) registerTools() {
	s.AddTool(skills.Tool{
		Name:        "web_search",
		Description: "Search the web for current information, news, facts, or any topic. Use this when you need up-to-date information that might not be in your training data.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query - be specific and include key terms",
				},
				"num_results": map[string]interface{}{
					"type":        "integer",
					"description": "Number of results to return (default: 5, max: 20)",
				},
				"provider": map[string]interface{}{
					"type":        "string",
					"description": "Search provider to use (brave, serper, google, duckduckgo). Leave empty for default.",
					"enum":        []string{"", "brave", "serper", "google", "duckduckgo"},
				},
			},
			"required": []string{"query"},
		},
		Handler: s.handleWebSearch,
	})

	s.AddTool(skills.Tool{
		Name:        "get_search_providers",
		Description: "Get information about available search providers and their status",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: s.handleGetProviders,
	})
}

func (s *SearchSkill) handleWebSearch(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required")
	}

	// Get number of results
	numResults := s.config.MaxResults
	if nr, ok := args["num_results"].(float64); ok {
		numResults = int(nr)
		if numResults > 20 {
			numResults = 20
		}
		if numResults < 1 {
			numResults = 1
		}
	}

	// Get provider
	providerName := s.defaultProvider
	if p, ok := args["provider"].(string); ok && p != "" {
		providerName = p
	}

	// Get provider instance
	provider, ok := s.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("unknown search provider: %s", providerName)
	}

	if !provider.IsAvailable() {
		return nil, fmt.Errorf("search provider %s is not available (missing API key or configuration)", providerName)
	}

	// Perform search with timeout
	searchCtx, cancel := context.WithTimeout(ctx, time.Duration(s.config.TimeoutSecs)*time.Second)
	defer cancel()

	start := time.Now()
	response, err := provider.Search(searchCtx, query, numResults)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	response.SearchTime = time.Since(start)

	// Format results for better readability
	return s.formatSearchResults(response), nil
}

func (s *SearchSkill) handleGetProviders(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	providers := []map[string]interface{}{}

	for name, provider := range s.providers {
		providers = append(providers, map[string]interface{}{
			"name":       name,
			"available":  provider.IsAvailable(),
			"is_default": name == s.defaultProvider,
		})
	}

	return map[string]interface{}{
		"default_provider": s.defaultProvider,
		"providers":        providers,
	}, nil
}

func (s *SearchSkill) formatSearchResults(response *SearchResponse) map[string]interface{} {
	results := []map[string]interface{}{}

	for _, r := range response.Results {
		result := map[string]interface{}{
			"title":   r.Title,
			"url":     r.URL,
			"snippet": r.Snippet,
			"source":  r.Source,
		}
		if r.PublishedAt != "" {
			result["published_at"] = r.PublishedAt
		}
		results = append(results, result)
	}

	return map[string]interface{}{
		"query":          response.Query,
		"provider":       response.Provider,
		"total_results":  len(results),
		"search_time_ms": response.SearchTime.Milliseconds(),
		"results":        results,
	}
}

// ==================== BRAVE PROVIDER ====================

type BraveProvider struct {
	apiKey string
	client *http.Client
}

func NewBraveProvider(apiKey string) *BraveProvider {
	return &BraveProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *BraveProvider) Name() string { return "brave" }

func (p *BraveProvider) IsAvailable() bool {
	return p.apiKey != ""
}

func (p *BraveProvider) Search(ctx context.Context, query string, numResults int) (*SearchResponse, error) {
	if !p.IsAvailable() {
		return nil, fmt.Errorf("Brave API key not configured")
	}

	// Build URL
	u, _ := url.Parse("https://api.search.brave.com/res/v1/web/search")
	q := u.Query()
	q.Set("q", query)
	q.Set("count", fmt.Sprintf("%d", numResults))
	q.Set("offset", "0")
	q.Set("mkt", "en-US")
	q.Set("safesearch", "moderate")
	q.Set("freshness", "all")
	q.Set("text_decorations", "false")
	q.Set("spellcheck", "true")
	u.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Subscription-Token", p.apiKey)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Brave API returned status %d", resp.StatusCode)
	}

	// Parse response
	var result struct {
		Web struct {
			Results []struct {
				Title   string `json:"title"`
				URL     string `json:"url"`
				Snippet string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode Brave response: %w", err)
	}

	// Convert to SearchResponse
	searchResults := []SearchResult{}
	for _, r := range result.Web.Results {
		searchResults = append(searchResults, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Snippet,
			Source:  "Brave Search",
		})
	}

	return &SearchResponse{
		Query:        query,
		Results:      searchResults,
		TotalResults: len(searchResults),
		Provider:     p.Name(),
	}, nil
}

// ==================== SERPER PROVIDER ====================

type SerperProvider struct {
	apiKey string
	client *http.Client
}

func NewSerperProvider(apiKey string) *SerperProvider {
	return &SerperProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *SerperProvider) Name() string { return "serper" }

func (p *SerperProvider) IsAvailable() bool {
	return p.apiKey != ""
}

func (p *SerperProvider) Search(ctx context.Context, query string, numResults int) (*SearchResponse, error) {
	if !p.IsAvailable() {
		return nil, fmt.Errorf("Serper API key not configured")
	}

	// Build request body
	body := map[string]interface{}{
		"q":    query,
		"num":  numResults,
		"page": 1,
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://google.serper.dev/search", strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-API-KEY", p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Serper API returned status %d", resp.StatusCode)
	}

	var result struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
			Date    string `json:"date"`
		} `json:"organic"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode Serper response: %w", err)
	}

	searchResults := []SearchResult{}
	for _, r := range result.Organic {
		searchResults = append(searchResults, SearchResult{
			Title:       r.Title,
			URL:         r.Link,
			Snippet:     r.Snippet,
			PublishedAt: r.Date,
			Source:      "Serper (Google)",
		})
	}

	return &SearchResponse{
		Query:        query,
		Results:      searchResults,
		TotalResults: len(searchResults),
		Provider:     p.Name(),
	}, nil
}

// ==================== DUCKDUCKGO PROVIDER ====================

type DuckDuckGoProvider struct {
	client *http.Client
}

func NewDuckDuckGoProvider() *DuckDuckGoProvider {
	return &DuckDuckGoProvider{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *DuckDuckGoProvider) Name() string { return "duckduckgo" }

func (p *DuckDuckGoProvider) IsAvailable() bool {
	// DuckDuckGo doesn't require an API key for basic HTML scraping
	return true
}

func (p *DuckDuckGoProvider) Search(ctx context.Context, query string, numResults int) (*SearchResponse, error) {
	start := time.Now()

	// Build URL
	u, _ := url.Parse("https://html.duckduckgo.com/html/")
	q := u.Query()
	q.Set("q", query)
	q.Set("kl", "us-en")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DuckDuckGo returned status %d", resp.StatusCode)
	}

	// Parse HTML using goquery
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DuckDuckGo HTML: %w", err)
	}

	var results []SearchResult

	// DuckDuckGo HTML results are in .result elements
	doc.Find(".result").Each(func(i int, s *goquery.Selection) {
		if len(results) >= numResults {
			return
		}

		// Extract title and URL
		linkElem := s.Find(".result__a")
		title := strings.TrimSpace(linkElem.Text())
		href, exists := linkElem.Attr("href")
		if !exists {
			return
		}

		// DuckDuckGo uses redirect URLs, extract the real URL
		resultURL := p.extractRealURL(href)

		// Extract snippet
		snippet := strings.TrimSpace(s.Find(".result__snippet").Text())

		// Extract source/domain
		source := strings.TrimSpace(s.Find(".result__url").Text())
		if source == "" {
			// Try alternative selector
			source = strings.TrimSpace(s.Find(".result__hostname").Text())
		}

		if title != "" && resultURL != "" {
			results = append(results, SearchResult{
				Title:   title,
				URL:     resultURL,
				Snippet: snippet,
				Source:  source,
			})
		}
	})

	// If no results found with primary selector, try alternative
	if len(results) == 0 {
		doc.Find(".links_main").Each(func(i int, s *goquery.Selection) {
			if len(results) >= numResults {
				return
			}

			linkElem := s.Find("a")
			title := strings.TrimSpace(linkElem.Text())
			href, exists := linkElem.Attr("href")
			if !exists {
				return
			}

			resultURL := p.extractRealURL(href)
			snippet := strings.TrimSpace(s.Find(".result__snippet").Text())

			if title != "" && resultURL != "" {
				results = append(results, SearchResult{
					Title:   title,
					URL:     resultURL,
					Snippet: snippet,
					Source:  resultURL,
				})
			}
		})
	}

	return &SearchResponse{
		Query:        query,
		Results:      results,
		TotalResults: len(results),
		Provider:     p.Name(),
		SearchTime:   time.Since(start),
	}, nil
}

// extractRealURL extracts the actual URL from DuckDuckGo's redirect URL
func (p *DuckDuckGoProvider) extractRealURL(duckURL string) string {
	// DuckDuckGo URLs are like: /l/?kh=-1&uddg=https%3A%2F%2Fexample.com
	if strings.HasPrefix(duckURL, "/l/") {
		u, err := url.Parse(duckURL)
		if err != nil {
			return duckURL
		}
		uddg := u.Query().Get("uddg")
		if uddg != "" {
			decoded, err := url.QueryUnescape(uddg)
			if err == nil {
				return decoded
			}
		}
	}

	// If it's already a full URL, return as-is
	if strings.HasPrefix(duckURL, "http://") || strings.HasPrefix(duckURL, "https://") {
		return duckURL
	}

	return duckURL
}

// ==================== GOOGLE PROVIDER ====================

type GoogleProvider struct {
	apiKey string
	client *http.Client
}

func NewGoogleProvider(apiKey string) *GoogleProvider {
	return &GoogleProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *GoogleProvider) Name() string { return "google" }

func (p *GoogleProvider) IsAvailable() bool {
	return p.apiKey != ""
}

func (p *GoogleProvider) Search(ctx context.Context, query string, numResults int) (*SearchResponse, error) {
	if !p.IsAvailable() {
		return nil, fmt.Errorf("Google Custom Search API key not configured")
	}

	// Note: Google Custom Search requires both API key and Search Engine ID (cx)
	// The cx parameter should be set via environment variable: GOOGLE_SEARCH_CX
	cx := os.Getenv("GOOGLE_SEARCH_CX")
	if cx == "" {
		return nil, fmt.Errorf("Google Custom Search requires Search Engine ID (cx). Set GOOGLE_SEARCH_CX environment variable")
	}

	start := time.Now()

	// Build URL
	u, _ := url.Parse("https://www.googleapis.com/customsearch/v1")
	q := u.Query()
	q.Set("key", p.apiKey)
	q.Set("cx", cx)
	q.Set("q", query)
	q.Set("num", fmt.Sprintf("%d", min(numResults, 10))) // Google max is 10 per request
	u.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Google API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		Items []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
			Pagemap struct {
				Metatags []struct {
					Date string `json:"date"`
				} `json:"metatags"`
			} `json:"pagemap"`
		} `json:"items"`
		SearchInformation struct {
			TotalResults string `json:"totalResults"`
		} `json:"searchInformation"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode Google response: %w", err)
	}

	// Convert to SearchResponse
	searchResults := []SearchResult{}
	for _, r := range result.Items {
		publishedAt := ""
		if len(r.Pagemap.Metatags) > 0 {
			publishedAt = r.Pagemap.Metatags[0].Date
		}
		searchResults = append(searchResults, SearchResult{
			Title:       r.Title,
			URL:         r.Link,
			Snippet:     r.Snippet,
			PublishedAt: publishedAt,
			Source:      "Google",
		})
	}

	totalResults := 0
	fmt.Sscanf(result.SearchInformation.TotalResults, "%d", &totalResults)

	return &SearchResponse{
		Query:        query,
		Results:      searchResults,
		TotalResults: totalResults,
		Provider:     p.Name(),
		SearchTime:   time.Since(start),
	}, nil
}
