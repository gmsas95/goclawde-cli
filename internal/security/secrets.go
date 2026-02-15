package security

import (
	"regexp"
)

type SecretMatch struct {
	Type        string
	Pattern     string
	Start       int
	End         int
	Redacted    string
}

type SecretScanner struct {
	patterns []*secretPattern
}

type secretPattern struct {
	name       string
	regex      *regexp.Regexp
	redactWith string
}

var defaultSecretPatterns = []struct {
	name       string
	pattern    string
	redactWith string
}{
	{"AWS Access Key", `AKIA[0-9A-Z]{16}`, "AKIA****"},
	{"AWS Secret Key", `(?i)aws(.{0,20})?['\"][0-9a-zA-Z/+=]{40}['\"]`, "AWS_SECRET****"},
	{"GitHub Token", `ghp_[0-9a-zA-Z]{36}`, "ghp_****"},
	{"GitHub OAuth", `gho_[0-9a-zA-Z]{36}`, "gho_****"},
	{"GitHub App Token", `(ghu|ghs)_[0-9a-zA-Z]{36}`, "gh*_****"},
	{"Slack Token", `xox[baprs]-[0-9]{10,13}-[0-9]{10,13}-[a-zA-Z0-9]{24}`, "xox*-****"},
	{"Slack Webhook", `https://hooks.slack.com/services/T[0-9A-Z]{8,12}/B[0-9A-Z]{8,12}/[0-9a-zA-Z]{24}`, "https://hooks.slack.com/****"},
	{"Stripe Key", `sk_live_[0-9a-zA-Z]{24}`, "sk_live_****"},
	{"Stripe Publishable", `pk_live_[0-9a-zA-Z]{24}`, "pk_live_****"},
	{"Google API Key", `AIza[0-9A-Za-z\-_]{35}`, "AIza****"},
	{"OpenAI API Key", `sk-[a-zA-Z0-9]{20}T3BlbkFJ[a-zA-Z0-9]{20}`, "sk-****"},
	{"Generic API Key", `(?i)(api[_-]?key|apikey|access[_-]?key)['\"]?\s*[:=]\s*['\"]?[0-9a-zA-Z\-_]{20,}['\"]?`, "API_KEY****"},
	{"Private Key", `-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`, "PRIVATE_KEY****"},
	{"JWT Token", `eyJ[a-zA-Z0-9\-_]+\.eyJ[a-zA-Z0-9\-_]+\.[a-zA-Z0-9\-_]+`, "eyJ****"},
	{"Generic Secret", `(?i)(secret|password|passwd|pwd|token)['\"]?\s*[:=]\s*['\"]?[^\s'\"]{8,}['\"]?`, "SECRET****"},
	{"Telegram Bot Token", `[0-9]{8,10}:[a-zA-Z0-9_-]{35}`, "****:****"},
	{"Discord Token", `[MN][a-zA-Z\d]{23}\.[\w-]{6}\.[\w-]{27}`, "DISCORD_TOKEN****"},
	{"Database URL", `(?i)(postgres|mysql|mongodb|redis)://[^\s'\"]+:[^\s'\"]+@[^\s'\"]+`, "DB_URL****"},
}

func NewSecretScanner() *SecretScanner {
	scanner := &SecretScanner{
		patterns: make([]*secretPattern, 0, len(defaultSecretPatterns)),
	}

	for _, p := range defaultSecretPatterns {
		re, err := regexp.Compile(p.pattern)
		if err == nil {
			scanner.patterns = append(scanner.patterns, &secretPattern{
				name:       p.name,
				regex:      re,
				redactWith: p.redactWith,
			})
		}
	}

	return scanner
}

func (s *SecretScanner) Scan(input string) []SecretMatch {
	var matches []SecretMatch

	for _, pattern := range s.patterns {
		locs := pattern.regex.FindAllStringIndex(input, -1)
		for _, loc := range locs {
			matches = append(matches, SecretMatch{
				Type:     pattern.name,
				Pattern:  pattern.regex.String(),
				Start:    loc[0],
				End:      loc[1],
				Redacted: pattern.redactWith,
			})
		}
	}

	return matches
}

func (s *SecretScanner) HasSecrets(input string) bool {
	return len(s.Scan(input)) > 0
}

func (s *SecretScanner) Redact(input string) string {
	result := input

	for _, pattern := range s.patterns {
		result = pattern.regex.ReplaceAllString(result, pattern.redactWith)
	}

	return result
}

func ScanForSecrets(input string) []SecretMatch {
	return NewSecretScanner().Scan(input)
}

func HasSecrets(input string) bool {
	return NewSecretScanner().HasSecrets(input)
}

func RedactSecrets(input string) string {
	return NewSecretScanner().Redact(input)
}
