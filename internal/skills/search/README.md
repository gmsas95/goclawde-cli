# Web Search Skill

The Web Search skill provides real-time information retrieval from the internet using multiple search providers.

## Features

- **Multiple Search Providers**: Support for Brave Search, Serper (Google), DuckDuckGo, and Google Custom Search
- **Real-time Information**: Access current news, events, and data beyond your LLM's training cutoff
- **Automatic Provider Selection**: Uses the first available provider by default
- **Configurable**: Easy configuration via environment variables or config file

## Configuration

### Environment Variables

```bash
# Enable web search (default: true)
export MYRAI_SEARCH_ENABLED=true

# Choose provider: brave, serper, duckduckgo, google (default: brave)
export MYRAI_SEARCH_PROVIDER=brave

# API key for your chosen provider
export MYRAI_SEARCH_API_KEY=your_api_key_here

# Optional: Max results per search (default: 5)
export MYRAI_SEARCH_MAX_RESULTS=10

# Optional: Timeout in seconds (default: 30)
export MYRAI_SEARCH_TIMEOUT_SECONDS=30
```

### Config File (myrai.yaml)

```yaml
search:
  enabled: true
  provider: brave
  api_key: your_api_key_here
  max_results: 5
  timeout_seconds: 30
```

## Search Providers

### Brave Search (Recommended)
- **Website**: https://search.brave.com/api
- **Free Tier**: 2,000 queries/month
- **Pros**: Privacy-focused, fast, high-quality results
- **Cons**: API key required

### Serper (Google Search API)
- **Website**: https://serper.dev
- **Free Tier**: 2,500 queries
- **Pros**: Google-quality results, includes rich snippets
- **Cons**: API key required

### DuckDuckGo
- **Website**: https://duckduckgo.com
- **Free Tier**: Unlimited (HTML scraping)
- **Pros**: No API key needed, privacy-focused
- **Cons**: Less reliable, may be rate-limited

### Google Custom Search
- **Website**: https://developers.google.com/custom-search
- **Free Tier**: 100 queries/day
- **Pros**: Google's algorithm
- **Cons**: Requires both API key and Search Engine ID, strict limits

## Usage

Once configured, the AI will automatically use web search when it needs current information. You can also explicitly ask:

```
User: "What are the latest developments in AI?"
AI: *uses web_search tool to find current information*

User: "Search for the current weather in Tokyo"
AI: *uses web_search tool*
```

### Tools Available

1. **web_search**: Search the web for information
   - Parameters:
     - `query` (required): Search query
     - `num_results`: Number of results (default: 5, max: 20)
     - `provider`: Specific provider to use (optional)

2. **get_search_providers**: List available search providers
   - Shows which providers are configured and available

## When Web Search is Used

The AI is instructed to use web search for:
- Current events and news
- Recent developments
- Time-sensitive information (weather, stock prices, sports)
- Facts that may have changed recently
- Queries containing "latest", "current", "recent", "today", "news"

## Examples

```
# News and current events
"What's happening in the news today?"
"Latest updates on climate change"

# Time-sensitive information
"Current Bitcoin price"
"Weather forecast for this weekend"

# Recent developments
"New features in Go 1.22"
"Recent AI breakthroughs"

# Fact checking
"Who won the World Cup 2022?"
"Current president of France"
```

## Troubleshooting

### "search provider X is not available"
- Check that you've set the correct API key
- Verify the provider name in your config
- Check provider-specific requirements

### No results returned
- Try a different search provider
- Check your internet connection
- Verify your API key is valid and has quota remaining

### Slow searches
- Increase timeout in config: `timeout_seconds: 60`
- Try a different provider
- Check your network connection

## Privacy

- Search queries are sent to the configured search provider
- No search history is stored by Myrai (only results are cached temporarily)
- Use DuckDuckGo provider for maximum privacy (no API key needed)

## API Key Setup

### Getting a Brave Search API Key
1. Go to https://api.search.brave.com/
2. Sign up for an account
3. Generate an API key
4. Set `MYRAI_SEARCH_API_KEY=your_key`

### Getting a Serper API Key
1. Go to https://serper.dev
2. Create an account
3. Copy your API key
4. Set `MYRAI_SEARCH_API_KEY=your_key`

## Development

The search skill is located at `internal/skills/search/`.

To add a new provider:
1. Implement the `Provider` interface
2. Register it in `registerProviders()`
3. Add configuration options in `Config`
4. Update documentation
