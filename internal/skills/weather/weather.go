package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/YOUR_USERNAME/jimmy.ai/internal/skills"
)

// WeatherSkill provides weather information
type WeatherSkill struct {
	*skills.BaseSkill
	apiKey string
}

// NewWeatherSkill creates a new weather skill
// Uses wttr.in which doesn't require API key
func NewWeatherSkill() *WeatherSkill {
	s := &WeatherSkill{
		BaseSkill: skills.NewBaseSkill("weather", "Weather information", "1.0.0"),
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
				"units": map[string]interface{}{
					"type":        "string",
					"description": "Units: metric or imperial",
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

	// Use wttr.in which is free and doesn't require API key
	url := fmt.Sprintf("https://wttr.in/%s?format=j1", location)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch weather: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather service returned: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	// Parse current condition
	current, ok := result["current_condition"].([]interface{})
	if !ok || len(current) == 0 {
		return nil, fmt.Errorf("no weather data available")
	}

	condition := current[0].(map[string]interface{})

	// Parse location info
	nearest, ok := result["nearest_area"].([]interface{})
	if !ok || len(nearest) == 0 {
		return nil, fmt.Errorf("location not found")
	}
	area := nearest[0].(map[string]interface{})
	areaName := area["areaName"].([]interface{})[0].(map[string]interface{})["value"]
	country := area["country"].([]interface{})[0].(map[string]interface{})["value"]

	// Format response
	return map[string]interface{}{
		"location":    fmt.Sprintf("%s, %s", areaName, country),
		"temperature": condition["temp_C"],
		"feels_like":  condition["FeelsLikeC"],
		"description": condition["weatherDesc"].([]interface{})[0].(map[string]interface{})["value"],
		"humidity":    condition["humidity"],
		"wind":        condition["windspeedKmph"],
		"visibility":  condition["visibility"],
		"pressure":    condition["pressure"],
		"units":       "metric",
	}, nil
}

func (s *WeatherSkill) handleGetForecast(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	location, _ := args["location"].(string)
	if location == "" {
		return nil, fmt.Errorf("location is required")
	}

	days := 3
	if d, ok := args["days"].(float64); ok {
		days = int(d)
		if days > 5 {
			days = 5 // Limit to 5 days
		}
	}

	url := fmt.Sprintf("https://wttr.in/%s?format=j1", location)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch forecast: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	weather, ok := result["weather"].([]interface{})
	if !ok || len(weather) == 0 {
		return nil, fmt.Errorf("no forecast data available")
	}

	// Parse forecast for requested days
	forecast := make([]map[string]interface{}, 0, days)
	for i := 0; i < days && i < len(weather); i++ {
		day := weather[i].(map[string]interface{})
		
		// Get hourly data for averages
		hourly, _ := day["hourly"].([]interface{})
		var avgTemp float64
		var avgHumidity float64
		for _, h := range hourly {
			hour := h.(map[string]interface{})
			if temp, ok := hour["tempC"].(string); ok {
				var t float64
				fmt.Sscanf(temp, "%f", &t)
				avgTemp += t
			}
			if hum, ok := hour["humidity"].(string); ok {
				var h float64
				fmt.Sscanf(hum, "%f", &h)
				avgHumidity += h
			}
		}
		if len(hourly) > 0 {
			avgTemp /= float64(len(hourly))
			avgHumidity /= float64(len(hourly))
		}

		forecast = append(forecast, map[string]interface{}{
			"date":        day["date"],
			"max_temp":    day["maxtempC"],
			"min_temp":    day["mintempC"],
			"avg_temp":    avgTemp,
			"humidity":    avgHumidity,
			"description": day["hourly"].([]interface{})[0].(map[string]interface{})["weatherDesc"].([]interface{})[0].(map[string]interface{})["value"],
		})
	}

	return forecast, nil
}
