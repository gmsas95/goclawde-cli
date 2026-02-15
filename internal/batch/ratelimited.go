// Package batch provides rate-limited batch processing for high-throughput scenarios
package batch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimiterConfig holds rate limiting configuration
type RateLimiterConfig struct {
	// RPM - Requests Per Minute (0 = unlimited)
	RPM int
	
	// TPM - Tokens Per Minute (0 = unlimited)
	TPM int
	
	// MaxConcurrency - Maximum concurrent requests
	MaxConcurrency int
	
	// Burst size for rate limiter
	Burst int
}

// Tier3Config returns optimal config for Moonshot/Kimi Tier 3
func Tier3Config() RateLimiterConfig {
	return RateLimiterConfig{
		RPM:            5000,  // 5,000 requests per minute
		TPM:            3000000, // 3,000,000 tokens per minute
		MaxConcurrency: 200,   // 200 concurrent
		Burst:          100,   // Allow bursts of 100
	}
}

// Tier4Config returns optimal config for Moonshot/Kimi Tier 4
func Tier4Config() RateLimiterConfig {
	return RateLimiterConfig{
		RPM:            5000,
		TPM:            4000000, // 4,000,000 tokens per minute
		MaxConcurrency: 400,
		Burst:          100,
	}
}

// Tier5Config returns optimal config for Moonshot/Kimi Tier 5
func Tier5Config() RateLimiterConfig {
	return RateLimiterConfig{
		RPM:            10000, // 10,000 requests per minute
		TPM:            5000000, // 5,000,000 tokens per minute
		MaxConcurrency: 1000,
		Burst:          200,
	}
}

// RateLimitedProcessor extends Processor with rate limiting
type RateLimitedProcessor struct {
	*Processor
	rateLimiter   *rate.Limiter
	tokenLimiter  *rate.Limiter
	config        RateLimiterConfig
	tokensPerMin  int64
	requestsCount int64
	mu            sync.RWMutex
}

// NewRateLimitedProcessor creates a processor with rate limiting
func NewRateLimitedProcessor(base *Processor, rlConfig RateLimiterConfig) *RateLimitedProcessor {
	rp := &RateLimitedProcessor{
		Processor: base,
		config:    rlConfig,
	}

	// RPM limiter: convert to requests per second
	if rlConfig.RPM > 0 {
		rps := float64(rlConfig.RPM) / 60.0
		rp.rateLimiter = rate.NewLimiter(rate.Limit(rps), rlConfig.Burst)
	}

	// TPM limiter: convert to tokens per second
	if rlConfig.TPM > 0 {
		tps := float64(rlConfig.TPM) / 60.0
		rp.tokenLimiter = rate.NewLimiter(rate.Limit(tps), rlConfig.TPM/10) // 10% burst
	}

	return rp
}

// ProcessFileWithRateLimit processes with rate limiting and optimal concurrency
func (rp *RateLimitedProcessor) ProcessFileWithRateLimit(ctx context.Context, inputPath, outputPath string) (*Result, error) {
	startTime := time.Now()

	items, err := rp.loadInputFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load input file: %w", err)
	}

	result := &Result{
		Total:     len(items),
		StartTime: startTime,
		Items:     make([]OutputItem, 0, len(items)),
	}

	// Use optimal concurrency
	concurrency := rp.config.MaxConcurrency
	if concurrency > len(items) {
		concurrency = len(items)
	}

	rp.logger.Info("Starting rate-limited batch processing",
		zap.Int("total_items", len(items)),
		zap.Int("concurrency", concurrency),
		zap.Int("rpm_limit", rp.config.RPM),
		zap.Int("tpm_limit", rp.config.TPM),
	)

	// Progress tracking
	progress := &ProgressTracker{
		Total:     len(items),
		StartTime: startTime,
	}

	itemsChan := make(chan InputItem, len(items))
	resultsChan := make(chan OutputItem, len(items))

	// Start workers with rate limiting
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rp.rateLimitedWorker(ctx, workerID, itemsChan, resultsChan, progress)
		}(i)
	}

	// Feed items
	for _, item := range items {
		itemsChan <- item
	}
	close(itemsChan)

	// Collect results
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
		
		// Track tokens for TPM limiting
		rp.mu.Lock()
		rp.tokensPerMin += int64(output.TokensUsed)
		rp.requestsCount++
		rp.mu.Unlock()
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Log performance stats
	rp.logPerformanceStats(result)

	if outputPath != "" {
		if err := rp.saveOutputFile(outputPath, result); err != nil {
			return result, fmt.Errorf("failed to save output file: %w", err)
		}
	}

	return result, nil
}

func (rp *RateLimitedProcessor) rateLimitedWorker(ctx context.Context, workerID int, items <-chan InputItem, results chan<- OutputItem, progress *ProgressTracker) {
	for item := range items {
		// Wait for rate limiter
		if rp.rateLimiter != nil {
			if err := rp.rateLimiter.Wait(ctx); err != nil {
				results <- OutputItem{
					ID:      item.ID,
					Input:   item.Message,
					Error:   fmt.Sprintf("rate limit error: %v", err),
					Success: false,
				}
				continue
			}
		}

		output := rp.processItem(ctx, item)
		
		// Token-based rate limiting (approximate)
		if rp.tokenLimiter != nil && output.TokensUsed > 0 {
			// Reserve tokens (non-blocking, just tracks)
			rp.tokenLimiter.AllowN(time.Now(), output.TokensUsed)
		}

		results <- output
		progress.Increment()

		// Log progress every 100 items
		if progress.Completed%100 == 0 {
			rp.logger.Info("Batch progress",
				zap.Int("completed", progress.Completed),
				zap.Int("total", progress.Total),
				zap.Float64("percent", progress.Percent()),
				zap.Duration("elapsed", progress.Elapsed()),
				zap.Duration("eta", progress.ETA()),
			)
		}
	}
}

func (rp *RateLimitedProcessor) logPerformanceStats(result *Result) {
	duration := result.Duration.Minutes()
	if duration == 0 {
		duration = 0.001 // Avoid division by zero
	}

	rpm := float64(result.Success) / duration
	rp.mu.RLock()
	totalTokens := rp.tokensPerMin
	rp.mu.RUnlock()
	tpm := float64(totalTokens) / duration

	rp.logger.Info("Batch processing complete",
		zap.Int("total", result.Total),
		zap.Int("success", result.Success),
		zap.Int("failed", result.Failed),
		zap.Duration("duration", result.Duration),
		zap.Float64("rpm_actual", rpm),
		zap.Float64("tpm_actual", tpm),
		zap.Int64("total_tokens", totalTokens),
	)
}

// ProgressTracker tracks batch processing progress
type ProgressTracker struct {
	Total     int
	Completed int
	StartTime time.Time
	mu        sync.RWMutex
}

func (p *ProgressTracker) Increment() {
	p.mu.Lock()
	p.Completed++
	p.mu.Unlock()
}

func (p *ProgressTracker) Percent() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.Total == 0 {
		return 0
	}
	return float64(p.Completed) / float64(p.Total) * 100
}

func (p *ProgressTracker) Elapsed() time.Duration {
	return time.Since(p.StartTime)
}

func (p *ProgressTracker) ETA() time.Duration {
	p.mu.RLock()
	completed := p.Completed
	total := p.Total
	p.mu.RUnlock()

	if completed == 0 {
		return 0
	}

	elapsed := p.Elapsed()
	rate := float64(completed) / elapsed.Seconds()
	remaining := float64(total-completed) / rate
	
	return time.Duration(remaining) * time.Second
}

// ProcessInChunks processes large files in chunks with checkpointing
func (rp *RateLimitedProcessor) ProcessInChunks(ctx context.Context, inputPath, outputPath string, chunkSize int) ([]*Result, error) {
	items, err := rp.loadInputFile(inputPath)
	if err != nil {
		return nil, err
	}

	if len(items) <= chunkSize {
		result, err := rp.ProcessFileWithRateLimit(ctx, inputPath, outputPath)
		if err != nil {
			return nil, err
		}
		return []*Result{result}, nil
	}

	// Process in chunks
	var results []*Result
	chunks := (len(items) + chunkSize - 1) / chunkSize

	for i := 0; i < chunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(items) {
			end = len(items)
		}

		chunkItems := items[start:end]
		chunkFile := fmt.Sprintf("%s.chunk%d.jsonl", inputPath, i)
		chunkOutput := fmt.Sprintf("%s.chunk%d.json", outputPath, i)

		// Save chunk
		if err := rp.saveChunk(chunkFile, chunkItems); err != nil {
			return results, fmt.Errorf("failed to save chunk %d: %w", i, err)
		}

		rp.logger.Info("Processing chunk",
			zap.Int("chunk", i+1),
			zap.Int("total_chunks", chunks),
			zap.Int("items", len(chunkItems)),
		)

		result, err := rp.ProcessFileWithRateLimit(ctx, chunkFile, chunkOutput)
		if err != nil {
			rp.logger.Error("Chunk failed",
				zap.Int("chunk", i),
				zap.Error(err),
			)
			continue
		}

		results = append(results, result)
	}

	// Merge results
	mergedResult := rp.mergeResults(results)
	if outputPath != "" {
		rp.saveOutputFile(outputPath, mergedResult)
	}

	return results, nil
}

func (rp *RateLimitedProcessor) saveChunk(path string, items []InputItem) error {
	return rp.saveOutputFile(path, &Result{Items: make([]OutputItem, 0)})
}

func (rp *RateLimitedProcessor) mergeResults(results []*Result) *Result {
	merged := &Result{
		StartTime: results[0].StartTime,
		EndTime:   results[len(results)-1].EndTime,
	}

	for _, r := range results {
		merged.Total += r.Total
		merged.Success += r.Success
		merged.Failed += r.Failed
		merged.Skipped += r.Skipped
		merged.Items = append(merged.Items, r.Items...)
	}

	merged.Duration = merged.EndTime.Sub(merged.StartTime)
	return merged
}
