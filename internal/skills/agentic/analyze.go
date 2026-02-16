package agentic

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func (s *AgenticSkill) handleAnalyzeProjectStructure(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	depth := 3
	if d, ok := args["depth"].(float64); ok {
		depth = int(d)
	}

	result := map[string]interface{}{
		"path":         path,
		"languages":    detectLanguages(path),
		"frameworks":   detectFrameworks(path),
		"entry_points": findEntryPoints(path),
		"structure":    buildDirTree(path, depth),
	}

	result["file_counts"] = countFileTypes(path)

	return result, nil
}

func (s *AgenticSkill) handleAnalyzeCodeFile(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	result := map[string]interface{}{
		"path":       path,
		"size_bytes": len(content),
		"line_count": strings.Count(string(content), "\n") + 1,
		"todos":      extractTODOs(string(content)),
		"imports":    []string{},
		"functions":  []map[string]interface{}{},
		"classes":    []map[string]interface{}{},
	}

	if strings.HasSuffix(path, ".go") {
		analyzeGoFile(content, result)
	} else if strings.HasSuffix(path, ".py") {
		analyzePythonFile(string(content), result)
	} else if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".ts") {
		analyzeJSFile(string(content), result)
	}

	return result, nil
}

func (s *AgenticSkill) handleSearchCode(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}

	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	filePattern := "*"
	if fp, ok := args["file_pattern"].(string); ok && fp != "" {
		filePattern = fp
	}

	context := 2
	if c, ok := args["context"].(float64); ok {
		context = int(c)
	}

	cmd := exec.CommandContext(ctx, "sh", "-c",
		fmt.Sprintf("grep -rn --include=%q -C %d %q %s 2>/dev/null | head -100",
			filePattern, context, pattern, path))

	output, err := cmd.Output()
	if err != nil {
		return []map[string]interface{}{}, nil
	}

	return map[string]string{
		"results": string(output),
	}, nil
}

func (s *AgenticSkill) handleFindTodos(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	patterns := []string{"TODO", "FIXME", "HACK", "XXX", "BUG", "NOTE"}
	allResults := []map[string]string{}

	for _, pattern := range patterns {
		cmd := exec.CommandContext(ctx, "sh", "-c",
			fmt.Sprintf("grep -rn --include='*.go' --include='*.py' --include='*.js' --include='*.ts' --include='*.md' %q %s 2>/dev/null | head -20",
				pattern, path))

		output, _ := cmd.Output()
		lines := strings.Split(string(output), "\n")

		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 3)
			if len(parts) >= 3 {
				allResults = append(allResults, map[string]string{
					"type":    pattern,
					"file":    parts[0],
					"line":    parts[1],
					"content": strings.TrimSpace(parts[2]),
				})
			}
		}
	}

	return allResults, nil
}

func detectLanguages(path string) []string {
	languages := map[string]bool{}
	exts := map[string]string{
		".go":    "Go",
		".py":    "Python",
		".js":    "JavaScript",
		".ts":    "TypeScript",
		".jsx":   "React",
		".tsx":   "React TypeScript",
		".rs":    "Rust",
		".java":  "Java",
		".kt":    "Kotlin",
		".cpp":   "C++",
		".c":     "C",
		".h":     "C/C++",
		".rb":    "Ruby",
		".php":   "PHP",
		".swift": "Swift",
	}

	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := filepath.Ext(p)
		if lang, ok := exts[ext]; ok {
			languages[lang] = true
		}
		return nil
	})

	result := []string{}
	for lang := range languages {
		result = append(result, lang)
	}
	sort.Strings(result)
	return result
}

func detectFrameworks(path string) []string {
	frameworks := []string{}

	indicators := map[string]string{
		"go.mod":             "Go Modules",
		"package.json":       "Node.js",
		"requirements.txt":   "Python pip",
		"Cargo.toml":         "Rust Cargo",
		"pom.xml":            "Maven",
		"build.gradle":       "Gradle",
		"Dockerfile":         "Docker",
		"docker-compose.yml": "Docker Compose",
		".github":            "GitHub Actions",
		"main.go":            "Go CLI",
	}

	for file, framework := range indicators {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			frameworks = append(frameworks, framework)
		}
	}

	return frameworks
}

func findEntryPoints(path string) []string {
	entryPoints := []string{}

	patterns := []string{
		"main.go", "main.py", "index.js", "index.ts", "app.py",
		"cmd/*/main.go", "src/main.rs", "src/lib.rs",
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(filepath.Join(path, pattern))
		entryPoints = append(entryPoints, matches...)
	}

	return entryPoints
}

func buildDirTree(path string, maxDepth int) map[string]interface{} {
	result := map[string]interface{}{
		"name":  filepath.Base(path),
		"type":  "directory",
		"items": []map[string]interface{}{},
	}

	if maxDepth <= 0 {
		return result
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return result
	}

	items := []map[string]interface{}{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		item := map[string]interface{}{
			"name": entry.Name(),
			"type": "file",
		}

		if entry.IsDir() {
			item["type"] = "directory"
			if maxDepth > 1 {
				subPath := filepath.Join(path, entry.Name())
				item["items"] = buildDirTree(subPath, maxDepth-1)["items"]
			}
		}

		items = append(items, item)
	}

	result["items"] = items
	return result
}

func countFileTypes(path string) map[string]int {
	counts := map[string]int{}

	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := filepath.Ext(info.Name())
		if ext != "" {
			counts[ext]++
		}
		return nil
	})

	return counts
}

func extractTODOs(content string) []map[string]string {
	todos := []map[string]string{}
	pattern := regexp.MustCompile(`(?i)(TODO|FIXME|HACK|XXX|BUG|NOTE)[\s:]*(.+)`)

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if matches := pattern.FindStringSubmatch(line); matches != nil {
			todos = append(todos, map[string]string{
				"type":    strings.ToUpper(matches[1]),
				"content": strings.TrimSpace(matches[2]),
				"line":    strconv.Itoa(i + 1),
			})
		}
	}

	return todos
}

func analyzeGoFile(content []byte, result map[string]interface{}) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		return
	}

	imports := []string{}
	for _, imp := range f.Imports {
		if imp.Path != nil {
			imports = append(imports, strings.Trim(imp.Path.Value, `"`))
		}
	}
	result["imports"] = imports

	functions := []map[string]interface{}{}
	classes := []map[string]interface{}{}

	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			fn := map[string]interface{}{
				"name": x.Name.Name,
				"line": fset.Position(x.Pos()).Line,
			}
			if x.Recv != nil {
				fn["type"] = "method"
				for _, recv := range x.Recv.List {
					if len(recv.Names) > 0 {
						fn["receiver"] = recv.Names[0].Name
					}
				}
			} else {
				fn["type"] = "function"
			}
			functions = append(functions, fn)

		case *ast.TypeSpec:
			if _, ok := x.Type.(*ast.StructType); ok {
				classes = append(classes, map[string]interface{}{
					"name": x.Name.Name,
					"type": "struct",
					"line": fset.Position(x.Pos()).Line,
				})
			}
			if _, ok := x.Type.(*ast.InterfaceType); ok {
				classes = append(classes, map[string]interface{}{
					"name": x.Name.Name,
					"type": "interface",
					"line": fset.Position(x.Pos()).Line,
				})
			}
		}
		return true
	})

	result["functions"] = functions
	result["classes"] = classes
	result["package"] = f.Name.Name
}

func analyzePythonFile(content string, result map[string]interface{}) {
	functions := []map[string]interface{}{}
	classes := []map[string]interface{}{}
	imports := []string{}

	importPattern := regexp.MustCompile(`^(import|from)\s+(\S+)`)
	funcPattern := regexp.MustCompile(`^def\s+(\w+)\s*\(`)
	classPattern := regexp.MustCompile(`^class\s+(\w+)`)

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if matches := importPattern.FindStringSubmatch(line); matches != nil {
			imports = append(imports, matches[2])
		}
		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			functions = append(functions, map[string]interface{}{
				"name": matches[1],
				"type": "function",
				"line": i + 1,
			})
		}
		if matches := classPattern.FindStringSubmatch(line); matches != nil {
			classes = append(classes, map[string]interface{}{
				"name": matches[1],
				"type": "class",
				"line": i + 1,
			})
		}
	}

	result["imports"] = imports
	result["functions"] = functions
	result["classes"] = classes
}

func analyzeJSFile(content string, result map[string]interface{}) {
	functions := []map[string]interface{}{}
	classes := []map[string]interface{}{}
	imports := []string{}

	importPattern := regexp.MustCompile(`import\s+.*?\s+from\s+['"]([^'"]+)['"]`)
	funcPattern := regexp.MustCompile(`(?:async\s+)?function\s+(\w+)\s*\(`)
	classPattern := regexp.MustCompile(`class\s+(\w+)`)
	arrowFuncPattern := regexp.MustCompile(`(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\(`)

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if matches := importPattern.FindStringSubmatch(line); matches != nil {
			imports = append(imports, matches[1])
		}
		if matches := funcPattern.FindStringSubmatch(line); matches != nil {
			functions = append(functions, map[string]interface{}{
				"name": matches[1],
				"type": "function",
				"line": i + 1,
			})
		}
		if matches := classPattern.FindStringSubmatch(line); matches != nil {
			classes = append(classes, map[string]interface{}{
				"name": matches[1],
				"type": "class",
				"line": i + 1,
			})
		}
		if matches := arrowFuncPattern.FindStringSubmatch(line); matches != nil {
			functions = append(functions, map[string]interface{}{
				"name": matches[1],
				"type": "arrow_function",
				"line": i + 1,
			})
		}
	}

	result["imports"] = imports
	result["functions"] = functions
	result["classes"] = classes
}
