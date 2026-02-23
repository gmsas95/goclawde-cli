package marketplace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Badge types
const (
	BadgeVerified        = "verified"
	BadgeSecurityAudited = "security-audited"
	BadgeTested          = "tested"
	BadgeWellDocumented  = "well-documented"
	BadgeOfficial        = "official"
	BadgeCommunity       = "community"
	BadgeTrending        = "trending"
)

// VerificationResult holds the results of verification checks
type VerificationResult struct {
	Passed        bool            `json:"passed"`
	Score         int             `json:"score"`          // 0-100
	SecurityScore int             `json:"security_score"` // 0-100
	QualityScore  int             `json:"quality_score"`  // 0-100
	Badges        []string        `json:"badges"`
	Errors        []string        `json:"errors,omitempty"`
	Warnings      []string        `json:"warnings,omitempty"`
	Checks        map[string]bool `json:"checks"`
}

// Verifier handles agent package verification
type Verifier struct {
	logger *zap.Logger
}

// NewVerifier creates a new verifier
func NewVerifier(logger *zap.Logger) *Verifier {
	return &Verifier{
		logger: logger,
	}
}

// Verify performs all verification checks on an agent package
func (v *Verifier) Verify(ctx context.Context, pkg *AgentPackage, bundle *AgentPackageBundle) (*VerificationResult, error) {
	result := &VerificationResult{
		Checks: make(map[string]bool),
	}

	// 1. YAML Validation
	v.checkYAMLValidation(pkg, result)

	// 2. Required Fields
	v.checkRequiredFields(pkg, result)

	// 3. Version Validation
	v.checkVersionValidation(pkg, result)

	// 4. Security Scan
	v.checkSecurityScan(pkg, bundle, result)

	// 5. Dependency Check
	v.checkDependencies(pkg, result)

	// 6. Documentation Check
	v.checkDocumentation(bundle, result)

	// 7. Test Files Check
	v.checkTestFiles(bundle, result)

	// 8. License Check
	v.checkLicense(pkg, bundle, result)

	// Calculate overall scores
	result.calculateScores()

	// Assign badges
	result.assignBadges()

	// Determine overall pass/fail
	result.Passed = len(result.Errors) == 0 && result.Score >= 60

	return result, nil
}

// checkYAMLValidation validates the YAML structure
func (v *Verifier) checkYAMLValidation(pkg *AgentPackage, result *VerificationResult) {
	checkName := "yaml_validation"

	if pkg == nil {
		result.Errors = append(result.Errors, "Failed to parse AGENT.yaml")
		result.Checks[checkName] = false
		return
	}

	// Try to marshal and unmarshal to ensure valid structure
	data, err := yaml.Marshal(pkg)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid YAML structure: %v", err))
		result.Checks[checkName] = false
		return
	}

	var testPkg AgentPackage
	if err := yaml.Unmarshal(data, &testPkg); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("YAML round-trip failed: %v", err))
		result.Checks[checkName] = false
		return
	}

	result.Checks[checkName] = true
}

// checkRequiredFields checks that all required fields are present
func (v *Verifier) checkRequiredFields(pkg *AgentPackage, result *VerificationResult) {
	checkName := "required_fields"

	required := map[string]string{
		"name":              pkg.Name,
		"version":           pkg.Version,
		"author":            pkg.Author,
		"description":       pkg.Description,
		"license":           pkg.License,
		"min_myrai_version": pkg.Requirements.MinMyraiVersion,
	}

	var missing []string
	for field, value := range required {
		if value == "" {
			missing = append(missing, field)
		}
	}

	if len(missing) > 0 {
		result.Errors = append(result.Errors,
			fmt.Sprintf("Missing required fields: %s", strings.Join(missing, ", ")))
		result.Checks[checkName] = false
		return
	}

	result.Checks[checkName] = true
}

// checkVersionValidation validates semantic versions
func (v *Verifier) checkVersionValidation(pkg *AgentPackage, result *VerificationResult) {
	checkName := "version_validation"

	// Validate agent version
	if !IsValidSemver(pkg.Version) {
		result.Errors = append(result.Errors,
			fmt.Sprintf("Invalid semantic version for agent: %s", pkg.Version))
		result.Checks[checkName] = false
		return
	}

	// Validate minimum Myrai version
	if !IsValidSemver(pkg.Requirements.MinMyraiVersion) {
		result.Errors = append(result.Errors,
			fmt.Sprintf("Invalid semantic version for min_myrai_version: %s",
				pkg.Requirements.MinMyraiVersion))
		result.Checks[checkName] = false
		return
	}

	// Check external skill versions
	for _, skill := range pkg.Skills.External {
		if skill.Version != "" && !IsValidSemver(skill.Version) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Invalid semantic version for skill %s: %s",
					skill.Name, skill.Version))
		}
	}

	result.Checks[checkName] = true
}

// checkSecurityScan performs basic security checks
func (v *Verifier) checkSecurityScan(pkg *AgentPackage, bundle *AgentPackageBundle, result *VerificationResult) {
	checkName := "security_scan"
	securityIssues := 0

	// Check MCP server commands for potential issues
	for _, server := range pkg.MCPServers {
		// Check for shell interpreters
		if strings.Contains(server.Command, "bash") ||
			strings.Contains(server.Command, "sh") ||
			strings.Contains(server.Command, "zsh") {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("MCP server '%s' uses shell interpreter - review for security", server.Name))
			securityIssues++
		}

		// Check for sudo
		if strings.Contains(server.Command, "sudo") {
			result.Errors = append(result.Errors,
				fmt.Sprintf("MCP server '%s' uses sudo - not allowed for security", server.Name))
			securityIssues++
		}

		// Check for network access
		if strings.Contains(server.Command, "curl") ||
			strings.Contains(server.Command, "wget") {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("MCP server '%s' may access network - review for security", server.Name))
			securityIssues++
		}
	}

	// Check for suspicious environment variables
	for _, server := range pkg.MCPServers {
		for key, value := range server.Env {
			if strings.Contains(strings.ToUpper(key), "TOKEN") ||
				strings.Contains(strings.ToUpper(key), "SECRET") ||
				strings.Contains(strings.ToUpper(key), "KEY") ||
				strings.Contains(strings.ToUpper(key), "PASSWORD") {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("MCP server '%s' has env var '%s' - ensure it's not hardcoded",
						server.Name, key))
				if value != "" {
					result.Errors = append(result.Errors,
						fmt.Sprintf("MCP server '%s' has hardcoded secret in '%s'",
							server.Name, key))
					securityIssues++
				}
			}
		}
	}

	// Scan files for suspicious patterns
	if bundle != nil {
		files, _ := bundle.ListFiles()
		for _, file := range files {
			// Skip binary files
			if isBinaryFile(file) {
				continue
			}

			content, err := bundle.GetFile(file)
			if err != nil {
				continue
			}

			contentStr := string(content)

			// Check for hardcoded secrets
			if containsPattern(contentStr, []string{
				"password=", "api_key=", "secret=", "token=",
				"BEGIN PRIVATE KEY", "BEGIN RSA PRIVATE KEY",
			}) {
				result.Errors = append(result.Errors,
					fmt.Sprintf("File '%s' may contain hardcoded secrets", file))
				securityIssues++
			}

			// Check for dangerous operations
			if containsPattern(contentStr, []string{
				"os.RemoveAll(\"/\")", "rm -rf /", "format C:",
				"DROP TABLE", "DELETE FROM",
			}) {
				result.Errors = append(result.Errors,
					fmt.Sprintf("File '%s' contains potentially dangerous operations", file))
				securityIssues++
			}
		}
	}

	result.SecurityScore = 100 - (securityIssues * 10)
	if result.SecurityScore < 0 {
		result.SecurityScore = 0
	}

	if securityIssues > 0 {
		result.Checks[checkName] = false
	} else {
		result.Checks[checkName] = true
	}
}

// checkDependencies validates dependencies
func (v *Verifier) checkDependencies(pkg *AgentPackage, result *VerificationResult) {
	checkName := "dependency_check"

	// Check that all skills reference valid names
	validBuiltinSkills := map[string]bool{
		"search":       true,
		"weather":      true,
		"calendar":     true,
		"tasks":        true,
		"expenses":     true,
		"health":       true,
		"shopping":     true,
		"github":       true,
		"browser":      true,
		"system":       true,
		"voice":        true,
		"intelligence": true,
		"knowledge":    true,
		"agentic":      true,
	}

	for _, skill := range pkg.Skills.Builtin {
		if !validBuiltinSkills[skill.Name] {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Unknown builtin skill: %s", skill.Name))
		}
	}

	// Check for duplicate skill names
	skillNames := make(map[string]int)
	for _, skill := range pkg.Skills.Builtin {
		skillNames[skill.Name]++
	}
	for name, count := range skillNames {
		if count > 1 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Duplicate builtin skill: %s", name))
		}
	}

	result.Checks[checkName] = true
}

// checkDocumentation checks for proper documentation
func (v *Verifier) checkDocumentation(bundle *AgentPackageBundle, result *VerificationResult) {
	checkName := "documentation"
	score := 0

	if bundle == nil {
		result.Warnings = append(result.Warnings, "No bundle to check for documentation")
		result.Checks[checkName] = false
		return
	}

	// Check for README
	readmeFound := false
	for _, name := range []string{"README.md", "README", "readme.md", "readme"} {
		if _, err := bundle.GetFile(name); err == nil {
			readmeFound = true
			score += 30
			break
		}
	}

	if !readmeFound {
		result.Warnings = append(result.Warnings, "No README found")
	}

	// Check for LICENSE
	licenseFound := false
	for _, name := range []string{"LICENSE", "LICENSE.md", "LICENSE.txt", "license", "license.md"} {
		if _, err := bundle.GetFile(name); err == nil {
			licenseFound = true
			score += 20
			break
		}
	}

	if !licenseFound {
		result.Warnings = append(result.Warnings, "No LICENSE file found")
	}

	// Check for CHANGELOG
	changelogFound := false
	for _, name := range []string{"CHANGELOG.md", "CHANGELOG", "changelog.md", "changelog", "HISTORY.md"} {
		if _, err := bundle.GetFile(name); err == nil {
			changelogFound = true
			score += 20
			break
		}
	}

	if !changelogFound {
		result.Warnings = append(result.Warnings, "No CHANGELOG found")
	}

	// Check for examples directory
	if _, err := bundle.GetFile("examples/"); err == nil {
		score += 15
	}

	// Check for docs directory
	if _, err := bundle.GetFile("docs/"); err == nil {
		score += 15
	}

	result.QualityScore = score
	result.Checks[checkName] = readmeFound
}

// checkTestFiles checks for test coverage
func (v *Verifier) checkTestFiles(bundle *AgentPackageBundle, result *VerificationResult) {
	checkName := "test_files"

	if bundle == nil {
		result.Checks[checkName] = false
		return
	}

	files, _ := bundle.ListFiles()
	hasTests := false
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") ||
			strings.HasSuffix(file, ".test.js") ||
			strings.HasSuffix(file, ".spec.js") ||
			strings.HasSuffix(file, ".test.py") ||
			strings.Contains(file, "/tests/") ||
			strings.Contains(file, "/test/") {
			hasTests = true
			break
		}
	}

	result.Checks[checkName] = hasTests
	if !hasTests {
		result.Warnings = append(result.Warnings, "No test files found")
	}
}

// checkLicense validates the license
func (v *Verifier) checkLicense(pkg *AgentPackage, bundle *AgentPackageBundle, result *VerificationResult) {
	checkName := "license"

	validLicenses := map[string]bool{
		"MIT":          true,
		"Apache-2.0":   true,
		"BSD-3-Clause": true,
		"GPL-3.0":      true,
		"LGPL-3.0":     true,
		"MPL-2.0":      true,
		"ISC":          true,
		"Unlicense":    true,
		"Proprietary":  true,
	}

	if !validLicenses[pkg.License] {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Uncommon license: %s", pkg.License))
	}

	result.Checks[checkName] = true
}

// calculateScores calculates the overall scores
func (r *VerificationResult) calculateScores() {
	// Calculate base score from checks
	passed := 0
	total := len(r.Checks)
	for _, checkPassed := range r.Checks {
		if checkPassed {
			passed++
		}
	}

	if total > 0 {
		r.Score = (passed * 100) / total
	}

	// Adjust for security and quality
	if r.SecurityScore < 50 {
		r.Score -= 20
	}
	if r.QualityScore < 50 {
		r.Score -= 10
	}

	// Ensure within bounds
	if r.Score > 100 {
		r.Score = 100
	}
	if r.Score < 0 {
		r.Score = 0
	}
}

// assignBadges assigns badges based on verification results
func (r *VerificationResult) assignBadges() {
	// Verified badge - passed all checks
	if r.Passed {
		r.Badges = append(r.Badges, BadgeVerified)
	}

	// Security Audited badge - high security score
	if r.SecurityScore >= 80 {
		r.Badges = append(r.Badges, BadgeSecurityAudited)
	}

	// Tested badge - has tests
	if r.Checks["test_files"] {
		r.Badges = append(r.Badges, BadgeTested)
	}

	// Well Documented badge - high quality score
	if r.QualityScore >= 70 {
		r.Badges = append(r.Badges, BadgeWellDocumented)
	}
}

// Helper functions

func isBinaryFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	binaryExts := []string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico",
		".zip", ".tar", ".gz", ".bz2", ".7z", ".rar",
		".exe", ".dll", ".so", ".dylib",
		".bin", ".dat", ".db"}
	for _, b := range binaryExts {
		if ext == b {
			return true
		}
	}
	return false
}

func containsPattern(content string, patterns []string) bool {
	contentLower := strings.ToLower(content)
	for _, pattern := range patterns {
		if strings.Contains(contentLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// RunTests runs automated tests for an agent package
func (v *Verifier) RunTests(ctx context.Context, bundle *AgentPackageBundle) error {
	if bundle == nil || bundle.RootPath == "" {
		return fmt.Errorf("no bundle to test")
	}

	// Look for test scripts
	testScripts := []string{"test.sh", "test.py", "test.js", "Makefile"}
	for _, script := range testScripts {
		scriptPath := filepath.Join(bundle.RootPath, script)
		if _, err := os.Stat(scriptPath); err == nil {
			v.logger.Info("Found test script", zap.String("script", script))
			// In a real implementation, we'd run the test script
			// For now, just log it
			return nil
		}
	}

	// Look for Go tests
	if files, err := filepath.Glob(filepath.Join(bundle.RootPath, "*_test.go")); err == nil && len(files) > 0 {
		v.logger.Info("Found Go test files", zap.Int("count", len(files)))
		// In a real implementation, we'd run go test
		return nil
	}

	v.logger.Info("No automated tests found")
	return nil
}
