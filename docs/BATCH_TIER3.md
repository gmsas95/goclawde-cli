# Batch Processing with Moonshot/Kimi Tier 3 Limits

Your Tier 3 subscription gives you **massive** batch processing capacity:

| Metric | Your Limit | Practical Throughput |
|--------|-----------|---------------------|
| **RPM** | 5,000 | ~4,800 sustained |
| **TPM** | 3,000,000 | ~2.9M sustained |
| **Concurrency** | 200 | 200 parallel requests |
| **TPD** | Unlimited | Process millions/day |

---

## 🚀 Realistic Throughput Calculations

### Short Prompts (~500 tokens)
- API latency: ~1-2 seconds
- With 200 concurrency: **~6,000-12,000 requests/minute**
- But limited by RPM: **5,000/minute**

**Daily capacity**: 5,000 × 60 × 24 = **7.2 million requests/day**

### Medium Prompts (~2,000 tokens)
- API latency: ~3-5 seconds  
- TPM limit kicks in: 3M tokens/min = ~1,500 requests/min
- **Practical: ~1,500-2,000/minute**

**Daily capacity**: ~2M requests/day

### Long Prompts (~10,000 tokens)
- API latency: ~10-15 seconds
- TPM limit: 3M/min = ~300 requests/min
- **Practical: ~300/minute**

**Daily capacity**: ~430K requests/day

---

## 💰 Cost Estimation

Moonshot Pricing (Kimi K2.5):
- Input: ¥0.012 / 1K tokens (~$0.0017)
- Output: ¥0.048 / 1K tokens (~$0.0068)

### Example: Processing 100,000 Items

| Scenario | Avg Tokens | Total Cost | Time on Tier 3 |
|----------|-----------|-----------|----------------|
| Short Q&A (1K tokens) | 1,000 | ~$170 | ~20 minutes |
| Summarization (4K tokens) | 4,000 | ~$680 | ~1 hour |
| Code generation (10K tokens) | 10,000 | ~$1,700 | ~5.5 hours |

---

## 📋 Usage Examples

### 1. Quick Start with Tier 3 Optimized Settings

```bash
# Process 10,000 items with optimal settings for Tier 3
goclawde batch -i prompts.jsonl -o results.json \
  -c 200 \        # Max concurrency (your limit)
  -t 30           # 30 second timeout
```

### 2. Processing Large Files (100K+ items)

```bash
# For very large files, use chunking to enable resume
# This is not yet in CLI but can be done via API

# Split into 10K item chunks
split -l 10000 large_file.txt chunk_

# Process each chunk
for chunk in chunk_*; do
  goclawde batch -i "$chunk" -o "${chunk}.out.json" -c 200 -t 30
done

# Merge results
cat chunk_*.out.json > final_results.json
```

### 3. API Usage with Rate Limiting

```go
package main

import (
    "context"
    "github.com/gmsas95/goclawde-cli/internal/batch"
)

func main() {
    // Create base processor
    baseProcessor := batch.NewProcessor(agent, batch.Config{
        MaxConcurrency: 200,
        Timeout:        30 * time.Second,
        RetryCount:     2,
    }, logger)
    
    // Wrap with Tier 3 rate limiting
    processor := batch.NewRateLimitedProcessor(
        baseProcessor,
        batch.Tier3Config(), // Uses 200 concurrency, 5000 RPM, 3M TPM
    )
    
    // Process with rate limiting
    result, err := processor.ProcessFileWithRateLimit(
        context.Background(),
        "input.jsonl",
        "output.json",
    )
}
```

---

## 🎯 Real-World VPS Workloads

### Use Case 1: Content Farm (SEO Agency)

**Goal**: Generate 50,000 SEO articles/month

```bash
# Create 50,000 article prompts
cat > generate_prompts.py << 'EOF'
import json
topics = ["tech", "finance", "health", "travel", "food"]
for i in range(50000):
    topic = topics[i % len(topics)]
    print(json.dumps({
        "id": f"article_{i:05d}",
        "message": f"Write a 1000-word SEO article about {topic}. Include H2 headings, bullet points, and a conclusion."
    }))
EOF

python3 generate_prompts.py > articles.jsonl

# Process (takes ~10-12 hours on Tier 3)
goclawde batch -i articles.jsonl -o articles.json -c 200 -t 120

# Cost: ~50,000 × 2,000 tokens × $0.003 = ~$300
```

---

### Use Case 2: Data Labeling Service

**Goal**: Classify 1 million customer support tickets

```bash
# Each ticket: ~500 tokens input, ~50 tokens output
# Batch size: 1M tickets

# Process in 10 batches of 100K
for i in {1..10}; do
  echo "Processing batch $i..."
  goclawde batch -i "tickets_batch_$i.jsonl" \
    -o "classified_$i.json" \
    -c 200 -t 10
done

# Total time: ~3.5 hours
# Cost: ~1M × 550 tokens × $0.003 = ~$1,650
```

---

### Use Case 3: Code Review Pipeline

**Goal**: Review 100,000 code snippets for security issues

```bash
# Generate review prompts
find /repos -name "*.go" -o -name "*.py" | head -100000 | \
while read file; do
  echo "{\"id\": \"$file\", \"message\": \"Review for security issues: $(base64 -w 0 $file)\"}"
done > code_reviews.jsonl

# Process with higher timeout for long code
goclawde batch -i code_reviews.jsonl -o reviews.json -c 100 -t 60

# Time: ~20 minutes
# Cost: ~100K × 3K tokens × $0.003 = ~$900
```

---

### Use Case 4: Translation Service

**Goal**: Translate 500,000 product descriptions (EN → CN)

```bash
# Each description: ~300 tokens
cat products.jsonl | jq -r '. | {id, message: "Translate to Chinese: \(.description)"}' > translate.jsonl

# Process
goclawde batch -i translate.jsonl -o translated.json -c 200 -t 15

# Time: ~2 hours
# Cost: ~500K × 600 tokens × $0.003 = ~$900
```

---

### Use Case 5: Synthetic Training Data

**Goal**: Generate 10 million instruction-response pairs for fine-tuning

```bash
# This is a multi-day job - use chunking
# Each pair: ~1,000 tokens total

# Split into 100 chunks of 100K
split -l 100000 instructions.jsonl chunk_

# Process with resume capability (run in screen/tmux)
for chunk in chunk_*; do
  echo "Processing $chunk at $(date)"
  goclawde batch -i "$chunk" -o "${chunk}.out" -c 200 -t 20
  sleep 5  # Brief pause between chunks
done

# Total time: ~35 hours (spread across multiple days)
# Cost: ~10M × 1K tokens × $0.003 = ~$30,000
```

---

## 🔧 Optimizing for Your VPS

### Network Tuning

```bash
# /etc/sysctl.conf for high-connection workloads
net.ipv4.tcp_tw_reuse = 1
net.ipv4.ip_local_port_range = 1024 65535
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
```

### System Limits

```bash
# /etc/security/limits.conf
* soft nofile 1000000
* hard nofile 1000000
```

### Go Runtime Tuning

```bash
# Run with optimized settings
GOMAXPROCS=4 
GOMEMLIMIT=3GiB
go run ./cmd/goclawde
```

---

## 📊 Monitoring Your Batch Jobs

### Real-time Progress

```bash
# Watch progress in another terminal
watch -n 5 'ls -la output.json && wc -l output.json'

# Or use the API
while true; do
  curl -s http://localhost:8080/api/metrics | jq '.batch_progress'
  sleep 5
done
```

### Log Analysis

```bash
# Extract performance stats
grep "rpm_actual" /var/log/goclawde.log | tail -20

# Plot throughput over time
grep "Batch progress" /var/log/goclawde.log | \
  awk '{print $4, $6}' > progress.csv
```

---

## ⚠️ Best Practices

### 1. **Start Small**
```bash
# Test with 100 items first
goclawde batch -i test_100.jsonl -o test_out.json -c 50
```

### 2. **Use Chunking for 100K+ Items**
- Easier to resume
- Better error isolation
- Parallel processing across multiple VPS

### 3. **Monitor Costs**
```bash
# Track token usage
watch -n 10 'curl -s http://localhost:8080/api/metrics | jq .total_tokens'
```

### 4. **Handle Failures**
```bash
# Retry failed items only
jq '.items[] | select(.success == false)' output.json > failed.json
jq '.items[] | select(.success == false) | {id, message: .input}' failed.json > retry.json
goclawde batch -i retry.json -o retry_out.json -c 200
```

### 5. **Time Your Jobs**
- Run during off-peak hours for better latency
- Schedule with cron: `0 2 * * * goclawde batch ...`

---

## 💡 Pro Tips

1. **Batch similar-length prompts** together for predictable throughput
2. **Use streaming=False** for batch (already default) - faster
3. **Pre-filter inputs** to avoid wasting API calls on invalid data
4. **Set appropriate timeouts**:
   - Q&A: 10-15s
   - Summarization: 30s
   - Code generation: 60-120s
   - Long-form writing: 120-300s

---

## 🔥 Maximum Realistic Daily Throughput

With Tier 3 on a decent VPS:

| Metric | Value |
|--------|-------|
| **Requests/day** | ~3-5 million |
| **Tokens/day** | ~2-3 billion |
| **Cost/day** | ~$5,000-10,000 |
| **Data processed** | ~10-50 GB text |

**This is enterprise-grade throughput!** 🚀

---

## 📞 When to Upgrade

| Sign | Action |
|------|--------|
| RPM consistently hitting 5,000 | Upgrade to Tier 5 (10,000 RPM) |
| TPM hitting 3M | Upgrade to Tier 4 (4M) or Tier 5 (5M) |
| Need >1000 concurrency | Use multiple API keys/VPS |
| Cost > ¥20,000/month | Negotiate enterprise pricing |
