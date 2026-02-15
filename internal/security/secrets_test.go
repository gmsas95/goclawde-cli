package security

import (
	"testing"
)

func TestSecretScanner_AWSAccessKey(t *testing.T) {
	scanner := NewSecretScanner()
	inputs := []string{
		"AWS_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE",
		"key: AKIA1234567890ABCDEF",
		"AWS key AKIATEST1234567890 is here",
	}

	for _, input := range inputs {
		matches := scanner.Scan(input)
		if len(matches) == 0 {
			t.Errorf("AWS Access Key not detected: %s", input)
		}
	}
}

func TestSecretScanner_GitHubToken(t *testing.T) {
	scanner := NewSecretScanner()
	inputs := []string{
		"GITHUB_TOKEN=ghp_1234567890abcdefghijklmnopqrstuv",
		"token: ghp_ABCDEFghijklmnopqr123456789012345",
	}

	for _, input := range inputs {
		matches := scanner.Scan(input)
		if len(matches) == 0 {
			t.Errorf("GitHub Token not detected: %s", input)
		}
	}
}

func TestSecretScanner_SlackToken(t *testing.T) {
	scanner := NewSecretScanner()
	inputs := []string{
		"xoxb-TESTPLACEHOLDER-TESTPLACEHOLD-TestPlaceHolderTestPlace",
		"SLACK_TOKEN=xoxp-TESTPLACEHOLDER-TESTPLACEHOLD-TESTPLACEHOLD-Testplaceholderstringfortest",
	}

	for _, input := range inputs {
		matches := scanner.Scan(input)
		if len(matches) == 0 {
			t.Errorf("Slack Token not detected: %s", input)
		}
	}
}

func TestSecretScanner_StripeKey(t *testing.T) {
	scanner := NewSecretScanner()
	inputs := []string{
		"STRIPE_KEY=sk_test_TESTPLACEHOLDER12345678ab",
		"STRIPE_PUBLIC=pk_test_TESTPLACEHOLDER12345678ab",
	}

	for _, input := range inputs {
		matches := scanner.Scan(input)
		if len(matches) == 0 {
			t.Errorf("Stripe Key not detected: %s", input)
		}
	}
}

func TestSecretScanner_GoogleAPIKey(t *testing.T) {
	scanner := NewSecretScanner()
	input := "GOOGLE_API_KEY=AIzaSyDaGmWKa4JsXZ-HjGw7ISLn_3namBGewQe"
	matches := scanner.Scan(input)
	if len(matches) == 0 {
		t.Errorf("Google API Key not detected")
	}
}

func TestSecretScanner_OpenAIKey(t *testing.T) {
	scanner := NewSecretScanner()
	key := "sk-12345678901234567890" + "T3BlbkFJ" + "12345678901234567890"
	input := "OPENAI_API_KEY=" + key
	matches := scanner.Scan(input)
	if len(matches) == 0 {
		t.Error("OpenAI API Key not detected")
	}
}

func TestSecretScanner_PrivateKey(t *testing.T) {
	scanner := NewSecretScanner()
	inputs := []string{
		"-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...",
		"-----BEGIN EC PRIVATE KEY-----\nMHQCAQEEIB...",
		"-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAA...",
		"-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQE...",
	}

	for _, input := range inputs {
		matches := scanner.Scan(input)
		if len(matches) == 0 {
			t.Errorf("Private Key not detected in: %s", input[:50])
		}
	}
}

func TestSecretScanner_JWTToken(t *testing.T) {
	scanner := NewSecretScanner()
	input := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	matches := scanner.Scan(input)
	if len(matches) == 0 {
		t.Error("JWT Token not detected")
	}
}

func TestSecretScanner_GenericSecret(t *testing.T) {
	scanner := NewSecretScanner()
	inputs := []string{
		"password=MySecretPassword123!",
		"SECRET_KEY=abc123def456ghi789jkl",
		"token: super_secret_token_value",
	}

	for _, input := range inputs {
		matches := scanner.Scan(input)
		if len(matches) == 0 {
			t.Errorf("Generic secret not detected: %s", input)
		}
	}
}

func TestSecretScanner_TelegramBotToken(t *testing.T) {
	scanner := NewSecretScanner()
	input := "TELEGRAM_BOT_TOKEN=1234567890:ABCdefGHIjklMNOpqrsTUVwxyz-123456"
	matches := scanner.Scan(input)
	if len(matches) == 0 {
		t.Error("Telegram Bot Token not detected")
	}
}

func TestSecretScanner_DatabaseURL(t *testing.T) {
	scanner := NewSecretScanner()
	inputs := []string{
		"postgres://user:password@localhost:5432/mydb",
		"mysql://admin:secret123@db.example.com:3306/production",
		"mongodb://root:pass@mongo.example.com:27017/admin",
		"redis://:secretredis@redis.example.com:6379/0",
	}

	for _, input := range inputs {
		matches := scanner.Scan(input)
		if len(matches) == 0 {
			t.Errorf("Database URL not detected: %s", input)
		}
	}
}

func TestSecretScanner_NoSecrets(t *testing.T) {
	scanner := NewSecretScanner()
	inputs := []string{
		"Hello, world!",
		"This is a normal message without any secrets.",
		"The quick brown fox jumps over the lazy dog.",
		"SELECT * FROM users WHERE id = 1",
		"function hello() { console.log('hi'); }",
	}

	for _, input := range inputs {
		matches := scanner.Scan(input)
		if len(matches) > 0 {
			t.Errorf("False positive detected in: %s", input)
		}
	}
}

func TestSecretScanner_HasSecrets(t *testing.T) {
	scanner := NewSecretScanner()

	if scanner.HasSecrets("Hello world") {
		t.Error("False positive on plain text")
	}

	if !scanner.HasSecrets("password=secret123") {
		t.Error("Did not detect secret")
	}
}

func TestSecretScanner_Redact(t *testing.T) {
	scanner := NewSecretScanner()

	input := "AWS_KEY=AKIAIOSFODNN7EXAMPLE and token=ghp_1234567890abcdefghijklmnopqrstuv"
	redacted := scanner.Redact(input)

	if redacted == input {
		t.Error("Secrets were not redacted")
	}

	if containsAny(redacted, []string{"AKIAIOSFODNN7EXAMPLE", "ghp_1234567890abcdefghijklmnopqrstuv"}) {
		t.Error("Redaction did not remove secrets")
	}
}

func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if stringsContains(s, substr) {
			return true
		}
	}
	return false
}

func stringsContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSecretScanner_MultipleSecrets(t *testing.T) {
	scanner := NewSecretScanner()
	input := `
		AWS_KEY=AKIAIOSFODNN7EXAMPLE
		GITHUB_TOKEN=ghp_1234567890abcdefghijklmnopqrstuv
		password=SuperSecret123!
	`
	matches := scanner.Scan(input)
	if len(matches) < 2 {
		t.Errorf("Expected multiple secrets, got %d", len(matches))
	}
}

func TestSecretScanner_MatchLocations(t *testing.T) {
	scanner := NewSecretScanner()
	input := "key=AKIAIOSFODNN7EXAMPLE end"
	matches := scanner.Scan(input)

	if len(matches) == 0 {
		t.Fatal("No match found")
	}

	match := matches[0]
	if match.Start < 0 || match.End <= match.Start {
		t.Errorf("Invalid match location: start=%d, end=%d", match.Start, match.End)
	}

	if match.End > len(input) {
		t.Errorf("Match end exceeds input length")
	}
}

func TestSecretScanner_SecretType(t *testing.T) {
	scanner := NewSecretScanner()
	input := "AKIAIOSFODNN7EXAMPLE"
	matches := scanner.Scan(input)

	if len(matches) == 0 {
		t.Fatal("No match found")
	}

	if matches[0].Type != "AWS Access Key" {
		t.Errorf("Wrong secret type: %s", matches[0].Type)
	}
}

func TestHasSecrets_Helper(t *testing.T) {
	if HasSecrets("normal text") {
		t.Error("HasSecrets returned true for normal text")
	}

	if !HasSecrets("password=secret") {
		t.Error("HasSecrets returned false for text with secret")
	}
}

func TestRedactSecrets_Helper(t *testing.T) {
	input := "AKIAIOSFODNN7EXAMPLE"
	redacted := RedactSecrets(input)

	if redacted == input {
		t.Error("RedactSecrets did not redact")
	}
}

func TestSecretScanner_EmptyInput(t *testing.T) {
	scanner := NewSecretScanner()
	matches := scanner.Scan("")
	if len(matches) != 0 {
		t.Error("Empty input should have no matches")
	}
}

func TestSecretScanner_CaseInsensitive(t *testing.T) {
	scanner := NewSecretScanner()
	inputs := []string{
		"PASSWORD=secret",
		"Password=secret",
		"password=secret",
		"pAssWoRd=secret",
	}

	for _, input := range inputs {
		matches := scanner.Scan(input)
		if len(matches) == 0 {
			t.Errorf("Case variation not detected: %s", input)
		}
	}
}

func BenchmarkSecretScanner_Scan_NoSecrets(b *testing.B) {
	scanner := NewSecretScanner()
	input := "This is a normal message without any secrets."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner.Scan(input)
	}
}

func BenchmarkSecretScanner_Scan_WithSecrets(b *testing.B) {
	scanner := NewSecretScanner()
	input := "AWS_KEY=AKIAIOSFODNN7EXAMPLE and GITHUB_TOKEN=ghp_1234567890abcdefghijklmnopqrstuv"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner.Scan(input)
	}
}

func BenchmarkSecretScanner_Redact(b *testing.B) {
	scanner := NewSecretScanner()
	input := "AWS_KEY=AKIAIOSFODNN7EXAMPLE"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner.Redact(input)
	}
}
