package security

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrCommandBlocked      = errors.New("command blocked: matches prohibited pattern")
	ErrSensitivePathAccess = errors.New("command blocked: contains sensitive path")
)

type ShellSecurityConfig struct {
	compiledPatterns []*regexp.Regexp
	literalPatterns  []string
	Enabled          bool
}

var defaultRegexPatterns = []string{
	`curl\s+.*\|\s*(sh|bash|zsh)`,
	`wget\s+.*\|\s*(sh|bash|zsh)`,
	`\|\s*(sh|bash|zsh)\s*$`,
	`bash\s+-i\s+>&\s*/dev/tcp`,
	`nc\s+.*-e\s+(sh|bash|/bin)`,
	`/dev/tcp/`,
	`/dev/udp/`,
	// Match rm -rf / or rm -rf /* - specifically root directory only
	`rm\s+(-[rf]{1,2}\s+)*(-[rf]{1,2}\s+)*/($|\s|;|\||&)`,
	`rm\s+(-[rf]{1,2}\s+)*(-[rf]{1,2}\s+)*/\*($|\s|;|\||&)`,
	`rm\s+(-[rf]{1,2}\s+)*(-[rf]{1,2}\s+)*/\s+#`,
	`mkfs(\.[a-z0-9]+)?\s`,
	`dd\s+.*if=/dev/(zero|random|urandom).*of=/dev/[sh]d`,
	`>\s*/dev/[sh]d[a-z]`,
	`chmod\s+(-R\s+)?777\s+/\s*$`,
	`chmod\s+(-R\s+)?777\s+/[a-z]`,
	`:\(\)\s*\{\s*:\|\s*:\s*&\s*\}\s*;\s*:\s*`,
	`fork\s*\(\s*\)`,
	`base64\s+(-d|--decode)`,
	`python[23]?\s+-c\s+`,
	`perl\s+-e\s+`,
	`ruby\s+-e\s+`,
	`node\s+-e\s+`,
	`\beval\s+`,
	`xargs\s+.*sh\b`,
	`xargs\s+.*bash\b`,
	`\benv\b.*\|\s*\w+`,
	`\bprintenv\b.*\|\s*\w+`,
	`\benv\b.*>\s*/`,
	`\bprintenv\b.*>\s*/`,
}

var defaultLiteralPatterns = []string{
	"/etc/shadow",
	"/etc/passwd",
	"~/.ssh/",
	".ssh/id_rsa",
	".ssh/id_ed25519",
	".ssh/id_ecdsa",
	".ssh/id_dsa",
	".ssh/authorized_keys",
	".aws/credentials",
	".kube/config",
}

func NewShellSecurityConfig() *ShellSecurityConfig {
	config := &ShellSecurityConfig{
		compiledPatterns: make([]*regexp.Regexp, 0, len(defaultRegexPatterns)),
		literalPatterns:  make([]string, 0, len(defaultLiteralPatterns)),
		Enabled:          true,
	}

	for _, pattern := range defaultRegexPatterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err == nil {
			config.compiledPatterns = append(config.compiledPatterns, re)
		}
	}

	for _, literal := range defaultLiteralPatterns {
		config.literalPatterns = append(config.literalPatterns, strings.ToLower(literal))
	}

	return config
}

func PermissiveShellConfig() *ShellSecurityConfig {
	return &ShellSecurityConfig{
		compiledPatterns: nil,
		literalPatterns:  nil,
		Enabled:          false,
	}
}

func (c *ShellSecurityConfig) BlockPattern(pattern string) error {
	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return err
	}
	c.compiledPatterns = append(c.compiledPatterns, re)
	return nil
}

func (c *ShellSecurityConfig) BlockLiteral(literal string) {
	c.literalPatterns = append(c.literalPatterns, strings.ToLower(literal))
}

func (c *ShellSecurityConfig) ValidateCommand(command string) error {
	if !c.Enabled {
		return nil
	}

	commandLower := strings.ToLower(command)

	for _, pattern := range c.compiledPatterns {
		if pattern.MatchString(command) {
			return ErrCommandBlocked
		}
	}

	for _, literal := range c.literalPatterns {
		if strings.Contains(commandLower, literal) {
			return ErrSensitivePathAccess
		}
	}

	return nil
}

func ValidateCommand(command string) error {
	return NewShellSecurityConfig().ValidateCommand(command)
}

func IsSafeCommand(command string) bool {
	return ValidateCommand(command) == nil
}
