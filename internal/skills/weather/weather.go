package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/gmsas95/goclawde-cli/internal/skills"
)

// WeatherSkill provides weather information
type WeatherSkill struct {
	*skills.BaseSkill
}

// NewWeatherSkill creates a new weather skill
func NewWeatherSkill() *WeatherSkill {
	s := &WeatherSkill{
		BaseSkill: skills.NewBaseSkill("weather", "Weather information via wttr.in or Open-Meteo", "1.0.0"),
	}
	s.registerTools()
	return s
}

func (s *WeatherSkill) registerTools() {
	s.AddTool(skills.Tool{
		Name:        "get_weather",
		Description: "Get current weather for a location",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"location": map[string]interface{}{
					"type":        "string",
					"description": "City name or location",
				},
			},
			"required": []string{"location"},
		},
		Handler: s.handleGetWeather,
	})

	s.AddTool(skills.Tool{
		Name:        "get_forecast",
		Description: "Get weather forecast for a location",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"location": map[string]interface{}{
					"type":        "string",
					"description": "City name or location",
				},
				"days": map[string]interface{}{
					"type":        "integer",
					"description": "Number of days (default: 3)",
				},
			},
			"required": []string{"location"},
		},
		Handler: s.handleGetForecast,
	})
}

func (s *WeatherSkill) handleGetWeather(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	location, _ := args["location"].(string)
	if location == "" {
		return nil, fmt.Errorf("location is required")
	}

	// Try wttr.in first (simple format)
	wttrURL := fmt.Sprintf("wttr.in/%s?format=3", sanitizeLocation(location))
	output, err := exec.CommandContext(ctx, "curl", "-s", "--max-time", "10", wttrURL).Output()
	
	if err == nil && len(output) > 0 && !strings.Contains(string(output), "ERROR") {
		return map[string]string{
			"weather": strings.TrimSpace(string(output)),
			"source":  "wttr.in",
		}, nil
	}

	// Fallback to Open-Meteo (more reliable JSON API)
	return s.getOpenMeteoCurrent(ctx, location)
}

func (s *WeatherSkill) getOpenMeteoCurrent(ctx context.Context, location string) (interface{}, error) {
	// First, geocode the location using Open-Meteo geocoding API
	geoURL := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1", url.QueryEscape(location))
	geoOutput, err := exec.CommandContext(ctx, "curl", "-s", "--max-time", "10", geoURL).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to geocode location: %w", err)
	}

	var geoResult struct {
		Results []struct {
			Name      string  `json:"name"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Country   string  `json:"country"`
		} `json:"results"`
	}
	
	if err := json.Unmarshal(geoOutput, &geoResult); err != nil {
		return nil, fmt.Errorf("failed to parse geocoding response")
	}
	
	if len(geoResult.Results) == 0 {
		return nil, fmt.Errorf("location not found: %s", location)
	}
	
	loc := geoResult.Results[0]
	
	// Now get weather data
	weatherURL := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current_weather=true",
		loc.Latitude, loc.Longitude)
	
	weatherOutput, err := exec.CommandContext(ctx, "curl", "-s", "--max-time", "10", weatherURL).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch weather: %w", err)
	}

	var weatherResult struct {
		CurrentWeather struct {
			Temperature float64 `json:"temperature"`
			Windspeed   float64 `json:"windspeed"`
			WeatherCode int     `json:"weathercode"`
			Time        string  `json:"time"`
		} `json:"current_weather"`
	}
	
	if err := json.Unmarshal(weatherOutput, &weatherResult); err != nil {
		return nil, fmt.Errorf("failed to parse weather response")
	}

	cw := weatherResult.CurrentWeather
	return map[string]interface{}{
		"location":    fmt.Sprintf("%s, %s", loc.Name, loc.Country),
		"temperature": fmt.Sprintf("%.1f°C", cw.Temperature),
		"windspeed":   fmt.Sprintf("%.1f km/h", cw.Windspeed),
		"condition":   weatherCodeToString(cw.WeatherCode),
		"source":      "Open-Meteo",
	}, nil
}

func (s *WeatherSkill) handleGetForecast(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	location, _ := args["location"].(string)
	if location == "" {
		return nil, fmt.Errorf("location is required")
	}

	days := 3
	if d, ok := args["days"].(float64); ok && d > 0 && d <= 7 {
		days = int(d)
	}

	// Try wttr.in first
	wttrURL := fmt.Sprintf("wttr.in/%s?%d", sanitizeLocation(location), days)
	output, err := exec.CommandContext(ctx, "curl", "-s", "--max-time", "10", wttrURL).Output()
	
	if err == nil && len(output) > 0 && !strings.Contains(string(output), "ERROR") {
		return map[string]string{
			"forecast": strings.TrimSpace(string(output)),
			"days":     fmt.Sprintf("%d", days),
			"source":   "wttr.in",
		}, nil
	}

	// Fallback to Open-Meteo
	return s.getOpenMeteoForecast(ctx, location, days)
}

func (s *WeatherSkill) getOpenMeteoForecast(ctx context.Context, location string, days int) (interface{}, error) {
	// Geocode first
	geoURL := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1", url.QueryEscape(location))
	geoOutput, err := exec.CommandContext(ctx, "curl", "-s", "--max-time", "10", geoURL).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to geocode location: %w", err)
	}

	var geoResult struct {
		Results []struct {
			Name      string  `json:"name"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Country   string  `json:"country"`
		} `json:"results"`
	}
	
	if err := json.Unmarshal(geoOutput, &geoResult); err != nil || len(geoResult.Results) == 0 {
		return nil, fmt.Errorf("location not found: %s", location)
	}
	
	loc := geoResult.Results[0]
	
	// Get forecast
	weatherURL := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&daily=temperature_2m_max,temperature_2m_min&timezone=auto&forecast_days=%d",
		loc.Latitude, loc.Longitude, days)
	
	weatherOutput, err := exec.CommandContext(ctx, "curl", "-s", "--max-time", "10", weatherURL).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch forecast: %w", err)
	}

	var result struct {
		Daily struct {
			Time        []string  `json:"time"`
			MaxTemp     []float64 `json:"temperature_2m_max"`
			MinTemp     []float64 `json:"temperature_2m_min"`
		} `json:"daily"`
	}
	
	if err := json.Unmarshal(weatherOutput, &result); err != nil {
		return nil, fmt.Errorf("failed to parse forecast response")
	}

	forecast := make([]map[string]string, 0, len(result.Daily.Time))
	for i := range result.Daily.Time {
		forecast = append(forecast, map[string]string{
			"date":     result.Daily.Time[i],
			"max_temp": fmt.Sprintf("%.1f°C", result.Daily.MaxTemp[i]),
			"min_temp": fmt.Sprintf("%.1f°C", result.Daily.MinTemp[i]),
		})
	}

	return map[string]interface{}{
		"location": fmt.Sprintf("%s, %s", loc.Name, loc.Country),
		"forecast": forecast,
		"days":     days,
		"source":   "Open-Meteo",
	}, nil
}

func sanitizeLocation(loc string) string {
	return strings.ReplaceAll(loc, " ", "+")
}

func weatherCodeToString(code int) string {
	codes := map[int]string{
		0:  "Clear sky",
		1:  "Mainly clear",
		2:  "Partly cloudy",
		3:  "Overcast",
		45: "Foggy",
		48: "Depositing rime fog",
		51: "Light drizzle",
		53: "Moderate drizzle",
		55: "Dense drizzle",
		61: "Slight rain",
		63: "Moderate rain",
		65: "Heavy rain",
		71: "Slight snow",
		73: "Moderate snow",
		75: "Heavy snow",
		95: "Thunderstorm",
	}
	if desc, ok := codes[code]; ok {
		return desc
	}
	return "Unknown"
}
