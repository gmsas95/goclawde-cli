package batch

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gmsas95/goclawde-cli/internal/agent"
	"github.com/gmsas95/goclawde-cli/internal/security"
)

type Processor struct {
	agent     *agent.Agent
	config    Config
	logger    interface {
		Info(msg string, fields ...interface{})
		Error(msg string, fields ...interface{})
	}
}

type Config struct {
	MaxConcurrency int
	Timeout        time.Duration
	RetryCount     int
	RetryDelay     time.Duration
	SkipInvalid    bool
	ValidateInput  bool
}

type InputItem struct {
	ID      string            `json:"id"`
	Message string            `json:"message"`
	Context map[string]string `json:"context,omitempty"`
}

type OutputItem struct {
	ID           string            `json:"id"`
	Input        string            `json:"input"`
	Response     string            `json:"response"`
	TokensUsed   int               `json:"tokens_used"`
	ResponseTime time.Duration     `json:"response_time"`
	Success      bool              `json:"success"`
	Error        string            `json:"error,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
}

type Result struct {
	Total     int
	Success   int
	Failed    int
	Skipped   int
	Duration  time.Duration
	Items     []OutputItem
	StartTime time.Time
	EndTime   time.Time
}

func DefaultConfig() Config {
	return Config{
		MaxConcurrency: 3,
		Timeout:        60 * time.Second,
		RetryCount:     2,
		RetryDelay:     1 * time.Second,
		SkipInvalid:    true,
		ValidateInput:  true,
	}
}

func NewProcessor(ag *agent.Agent, cfg Config, logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}) *Processor {
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 1
	}
	return &Processor{
		agent:  ag,
		config: cfg,
		logger: logger,
	}
}

func (p *Processor) ProcessFile(ctx context.Context, inputPath, outputPath string) (*Result, error) {
	startTime := time.Now()

	items, err := p.loadInputFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load input file: %w", err)
	}

	result := &Result{
		Total:     len(items),
		StartTime: startTime,
		Items:     make([]OutputItem, 0, len(items)),
	}

	itemsChan := make(chan InputItem, len(items))
	resultsChan := make(chan OutputItem, len(items))

	var wg sync.WaitGroup
	for i := 0; i < p.config.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.worker(ctx, itemsChan, resultsChan)
		}()
	}

	for _, item := range items {
		itemsChan <- item
	}
	close(itemsChan)

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for output := range resultsChan {
		result.Items = append(result.Items, output)
		if output.Success {
			result.Success++
		} else {
			if output.Error == "skipped" {
				result.Skipped++
			} else {
				result.Failed++
			}
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if outputPath != "" {
		if err := p.saveOutputFile(outputPath, result); err != nil {
			return result, fmt.Errorf("failed to save output file: %w", err)
		}
	}

	return result, nil
}

func (p *Processor) worker(ctx context.Context, items <-chan InputItem, results chan<- OutputItem) {
	for item := range items {
		output := p.processItem(ctx, item)
		results <- output
	}
}

func (p *Processor) processItem(ctx context.Context, item InputItem) OutputItem {
	output := OutputItem{
		ID:        item.ID,
		Input:     item.Message,
		Timestamp: time.Now(),
	}

	if p.config.ValidateInput {
		validation := security.ValidateUserInput(item.Message)
		if !validation.Valid {
			output.Error = "skipped"
			output.Warnings = validation.Errors
			output.Success = false
			return output
		}
		if len(validation.Warnings) > 0 {
			output.Warnings = validation.Warnings
		}
	}

	var resp *agent.ChatResponse
	var err error

	for attempt := 0; attempt <= p.config.RetryCount; attempt++ {
		processCtx, cancel := context.WithTimeout(ctx, p.config.Timeout)
		
		start := time.Now()
		resp, err = p.agent.Chat(processCtx, agent.ChatRequest{
			Message: item.Message,
			Stream:  false,
		})
		output.ResponseTime = time.Since(start)
		
		cancel()

		if err == nil {
			break
		}

		if attempt < p.config.RetryCount {
			time.Sleep(p.config.RetryDelay)
		}
	}

	if err != nil {
		output.Error = err.Error()
		output.Success = false
		return output
	}

	output.Response = resp.Content
	output.TokensUsed = resp.TokensUsed
	output.Success = true

	return output
}

func (p *Processor) loadInputFile(path string) ([]InputItem, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ext := strings.ToLower(path[len(path)-4:])
	if ext == "json" || strings.HasSuffix(strings.ToLower(path), ".jsonl") {
		return p.loadJSONFile(file)
	}

	return p.loadTextFile(file)
}

func (p *Processor) loadJSONFile(file *os.File) ([]InputItem, error) {
	var items []InputItem
	decoder := json.NewDecoder(file)
	
	for decoder.More() {
		var item InputItem
		if err := decoder.Decode(&item); err != nil {
			if p.config.SkipInvalid {
				continue
			}
			return nil, fmt.Errorf("failed to decode JSON: %w", err)
		}
		if item.ID == "" {
			item.ID = fmt.Sprintf("item-%d", len(items)+1)
		}
		items = append(items, item)
	}

	return items, nil
}

func (p *Processor) loadTextFile(file *os.File) ([]InputItem, error) {
	var items []InputItem
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		items = append(items, InputItem{
			ID:      fmt.Sprintf("line-%d", lineNum),
			Message: line,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return items, nil
}

func (p *Processor) saveOutputFile(path string, result *Result) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	ext := strings.ToLower(path[len(path)-4:])
	if ext == "json" {
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}

	for _, item := range result.Items {
		fmt.Fprintf(file, "=== %s ===\n", item.ID)
		fmt.Fprintf(file, "Input: %s\n", item.Input)
		fmt.Fprintf(file, "Response: %s\n", item.Response)
		if item.Error != "" {
			fmt.Fprintf(file, "Error: %s\n", item.Error)
		}
		fmt.Fprintf(file, "Tokens: %d | Time: %v\n\n", item.TokensUsed, item.ResponseTime)
	}

	return nil
}

func (r *Result) Summary() string {
	var sb strings.Builder
	sb.WriteString("=== Batch Processing Summary ===\n")
	sb.WriteString(fmt.Sprintf("Total:     %d\n", r.Total))
	sb.WriteString(fmt.Sprintf("Success:   %d\n", r.Success))
	sb.WriteString(fmt.Sprintf("Failed:    %d\n", r.Failed))
	sb.WriteString(fmt.Sprintf("Skipped:   %d\n", r.Skipped))
	sb.WriteString(fmt.Sprintf("Duration:  %v\n", r.Duration))
	return sb.String()
}

func (r *Result) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
