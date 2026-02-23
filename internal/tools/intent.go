package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gmsas95/myrai-cli/internal/llm"
)

// IntentType represents the type of user intent
type IntentType string

const (
	IntentTypeCommand      IntentType = "command"
	IntentTypeQuestion     IntentType = "question"
	IntentTypeTask         IntentType = "task"
	IntentTypeConversation IntentType = "conversation"
)

// IntentCategory represents the category of intent
type IntentCategory string

const (
	CategoryDevOps   IntentCategory = "devops"
	CategoryCoding   IntentCategory = "coding"
	CategoryResearch IntentCategory = "research"
	CategoryGeneral  IntentCategory = "general"
)

// ComplexityLevel represents task complexity
type ComplexityLevel string

const (
	ComplexitySimple   ComplexityLevel = "simple"
	ComplexityCompound ComplexityLevel = "compound"
	ComplexityComplex  ComplexityLevel = "complex"
)

// Intent represents a classified user intent
type Intent struct {
	Type       IntentType      `json:"type"`
	Category   IntentCategory  `json:"category"`
	Complexity ComplexityLevel `json:"complexity"`
	Confidence float64         `json:"confidence"`
	Keywords   []string        `json:"keywords"`
	RawInput   string          `json:"raw_input"`
}

// IntentClassifier uses LLM to classify user intents
type IntentClassifier struct {
	llmClient *llm.Client
}

// NewIntentClassifier creates a new intent classifier
func NewIntentClassifier(llmClient *llm.Client) *IntentClassifier {
	return &IntentClassifier{
		llmClient: llmClient,
	}
}

// Classify analyzes user input and returns intent classification
func (ic *IntentClassifier) Classify(ctx context.Context, input string) (*Intent, error) {
	if ic.llmClient == nil {
		// Fallback to rule-based classification if no LLM
		return ic.ruleBasedClassify(input), nil
	}

	systemPrompt := `You are an intent classifier. Analyze the user input and classify it into:
1. Type: command, question, task, or conversation
2. Category: devops, coding, research, or general
3. Complexity: simple (single action), compound (2-3 related actions), or complex (multi-step workflow)
4. Keywords: key terms extracted from the input
5. Confidence: 0.0 to 1.0

Respond in JSON format only:
{
  "type": "command|question|task|conversation",
  "category": "devops|coding|research|general",
  "complexity": "simple|compound|complex",
  "confidence": 0.95,
  "keywords": ["keyword1", "keyword2"]
}`

	response, err := ic.llmClient.SimpleChat(ctx, systemPrompt, input)
	if err != nil {
		return nil, fmt.Errorf("failed to classify intent: %w", err)
	}

	// Parse JSON response
	var intent Intent
	if err := json.Unmarshal([]byte(extractJSON(response)), &intent); err != nil {
		// Fallback to rule-based if parsing fails
		intent = *ic.ruleBasedClassify(input)
	}

	intent.RawInput = input

	// Validate and normalize
	intent.Type = normalizeIntentType(intent.Type)
	intent.Category = normalizeCategory(intent.Category)
	intent.Complexity = normalizeComplexity(intent.Complexity)

	if intent.Confidence < 0 || intent.Confidence > 1 {
		intent.Confidence = 0.5
	}

	return &intent, nil
}

// ruleBasedClassify provides fallback classification without LLM
func (ic *IntentClassifier) ruleBasedClassify(input string) *Intent {
	inputLower := strings.ToLower(input)
	intent := &Intent{
		RawInput:   input,
		Type:       IntentTypeConversation,
		Category:   CategoryGeneral,
		Complexity: ComplexitySimple,
		Confidence: 0.6,
		Keywords:   extractKeywords(input),
	}

	// Classify type
	if strings.HasSuffix(input, "?") || strings.HasPrefix(inputLower, "what") ||
		strings.HasPrefix(inputLower, "how") || strings.HasPrefix(inputLower, "why") ||
		strings.HasPrefix(inputLower, "when") || strings.HasPrefix(inputLower, "where") ||
		strings.HasPrefix(inputLower, "who") || strings.HasPrefix(inputLower, "can you explain") {
		intent.Type = IntentTypeQuestion
	} else if strings.Contains(inputLower, "deploy") || strings.Contains(inputLower, "build") ||
		strings.Contains(inputLower, "run") || strings.Contains(inputLower, "execute") ||
		strings.Contains(inputLower, "create") || strings.Contains(inputLower, "setup") {
		intent.Type = IntentTypeCommand
	} else if strings.Contains(inputLower, "workflow") || strings.Contains(inputLower, "process") ||
		strings.Contains(inputLower, "automate") || strings.Contains(inputLower, "chain") {
		intent.Type = IntentTypeTask
	}

	// Classify category
	if strings.Contains(inputLower, "docker") || strings.Contains(inputLower, "kubernetes") ||
		strings.Contains(inputLower, "k8s") || strings.Contains(inputLower, "deploy") ||
		strings.Contains(inputLower, "server") || strings.Contains(inputLower, "infra") ||
		strings.Contains(inputLower, "terraform") || strings.Contains(inputLower, "ansible") ||
		strings.Contains(inputLower, "ci/cd") || strings.Contains(inputLower, "pipeline") {
		intent.Category = CategoryDevOps
	} else if strings.Contains(inputLower, "code") || strings.Contains(inputLower, "function") ||
		strings.Contains(inputLower, "programming") || strings.Contains(inputLower, "debug") ||
		strings.Contains(inputLower, "refactor") || strings.Contains(inputLower, "test") ||
		strings.Contains(inputLower, "api") || strings.Contains(inputLower, "git") {
		intent.Category = CategoryCoding
	} else if strings.Contains(inputLower, "research") || strings.Contains(inputLower, "search") ||
		strings.Contains(inputLower, "find") || strings.Contains(inputLower, "analyze") ||
		strings.Contains(inputLower, "compare") || strings.Contains(inputLower, "investigate") {
		intent.Category = CategoryResearch
	}

	// Classify complexity
	wordCount := len(strings.Fields(input))
	actionCount := countActions(inputLower)

	if wordCount > 30 || actionCount > 3 {
		intent.Complexity = ComplexityComplex
	} else if wordCount > 15 || actionCount > 1 {
		intent.Complexity = ComplexityCompound
	}

	return intent
}

// extractJSON extracts JSON from a string that might contain markdown or other text
func extractJSON(s string) string {
	// Try to find JSON between braces
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}
	return s
}

// extractKeywords extracts important keywords from input
func extractKeywords(input string) []string {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "up": true, "about": true, "into": true,
		"through": true, "during": true, "before": true, "after": true, "above": true,
		"below": true, "between": true, "among": true, "is": true, "are": true, "was": true,
		"were": true, "be": true, "been": true, "being": true, "have": true, "has": true,
		"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
		"could": true, "should": true, "can": true, "may": true, "might": true,
	}

	words := strings.Fields(strings.ToLower(input))
	keywords := make([]string, 0)
	seen := make(map[string]bool)

	for _, word := range words {
		// Clean punctuation
		word = strings.Trim(word, ".,!?;:()[]{}\"'")
		if len(word) > 2 && !stopWords[word] && !seen[word] {
			keywords = append(keywords, word)
			seen[word] = true
			if len(keywords) >= 5 {
				break
			}
		}
	}

	return keywords
}

// countActions estimates the number of actions in the input
func countActions(input string) int {
	actionWords := []string{
		"build", "deploy", "run", "execute", "create", "setup", "install",
		"configure", "update", "delete", "remove", "push", "pull", "commit",
		"test", "debug", "fix", "refactor", "search", "find", "analyze",
	}

	count := 0
	for _, action := range actionWords {
		if strings.Contains(input, action) {
			count++
		}
	}
	return count
}

// normalizeIntentType ensures valid intent type
func normalizeIntentType(t IntentType) IntentType {
	switch t {
	case IntentTypeCommand, IntentTypeQuestion, IntentTypeTask, IntentTypeConversation:
		return t
	default:
		return IntentTypeConversation
	}
}

// normalizeCategory ensures valid category
func normalizeCategory(c IntentCategory) IntentCategory {
	switch c {
	case CategoryDevOps, CategoryCoding, CategoryResearch, CategoryGeneral:
		return c
	default:
		return CategoryGeneral
	}
}

// normalizeComplexity ensures valid complexity
func normalizeComplexity(c ComplexityLevel) ComplexityLevel {
	switch c {
	case ComplexitySimple, ComplexityCompound, ComplexityComplex:
		return c
	default:
		return ComplexitySimple
	}
}

// NeedsDecomposition returns true if the intent requires task decomposition
func (i *Intent) NeedsDecomposition() bool {
	return i.Complexity == ComplexityCompound || i.Complexity == ComplexityComplex
}

// String returns a string representation of the intent
func (i *Intent) String() string {
	return fmt.Sprintf("Intent{type=%s, category=%s, complexity=%s, confidence=%.2f}",
		i.Type, i.Category, i.Complexity, i.Confidence)
}
