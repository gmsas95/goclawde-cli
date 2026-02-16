package agentic

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func (s *AgenticSkill) handleGetSystemResources(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	result := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
		"cpus":      runtime.NumCPU(),
	}

	if output, err := exec.CommandContext(ctx, "cat", "/proc/cpuinfo").Output(); err == nil {
		result["cpu_info"] = parseCPUInfo(string(output))
	}

	if output, err := exec.CommandContext(ctx, "cat", "/proc/meminfo").Output(); err == nil {
		result["memory"] = parseMemInfo(string(output))
	}

	if output, err := exec.CommandContext(ctx, "df", "-h").Output(); err == nil {
		result["disk_usage"] = string(output)
	}

	if output, err := exec.CommandContext(ctx, "cat", "/proc/loadavg").Output(); err == nil {
		result["load_average"] = strings.TrimSpace(string(output))
	}

	if output, err := exec.CommandContext(ctx, "uptime", "-p").Output(); err == nil {
		result["uptime"] = strings.TrimSpace(string(output))
	}

	return result, nil
}

func (s *AgenticSkill) handleListProcesses(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	filter := ""
	if f, ok := args["filter"].(string); ok {
		filter = f
	}

	var cmd *exec.Cmd
	if filter != "" {
		cmd = exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("ps aux | grep -i %s | head -%d", filter, limit))
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("ps aux --sort=-%%mem | head -%d", limit+1))
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	processes := []map[string]string{}

	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 11 {
			processes = append(processes, map[string]string{
				"user":    fields[0],
				"pid":     fields[1],
				"cpu":     fields[2],
				"mem":     fields[3],
				"command": strings.Join(fields[10:], " "),
			})
		}
	}

	return processes, nil
}

func (s *AgenticSkill) handleGetNetworkInfo(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	result := map[string]interface{}{}

	if output, err := exec.CommandContext(ctx, "ip", "addr").Output(); err == nil {
		result["interfaces"] = string(output)
	} else if output, err := exec.CommandContext(ctx, "ifconfig").Output(); err == nil {
		result["interfaces"] = string(output)
	}

	if output, err := exec.CommandContext(ctx, "ip", "route").Output(); err == nil {
		result["routes"] = string(output)
	}

	if output, err := exec.CommandContext(ctx, "ss", "-tuln").Output(); err == nil {
		result["listening_ports"] = string(output)
	} else if output, err := exec.CommandContext(ctx, "netstat", "-tuln").Output(); err == nil {
		result["listening_ports"] = string(output)
	}

	if output, err := exec.CommandContext(ctx, "cat", "/etc/resolv.conf").Output(); err == nil {
		result["dns"] = string(output)
	}

	return result, nil
}

func (s *AgenticSkill) handleGetEnvironment(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	filter := ""
	if f, ok := args["filter"].(string); ok {
		filter = f
	}

	env := map[string]string{}
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			if filter == "" || strings.HasPrefix(parts[0], filter) {
				env[parts[0]] = parts[1]
			}
		}
	}

	env["GO_VERSION"] = runtime.Version()
	env["GO_OS"] = runtime.GOOS
	env["GO_ARCH"] = runtime.GOARCH

	return env, nil
}

func parseCPUInfo(data string) []map[string]string {
	cpus := []map[string]string{}
	current := map[string]string{}

	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if len(current) > 0 {
				cpus = append(cpus, current)
				current = map[string]string{}
			}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			current[key] = value
		}
	}

	return cpus
}

func parseMemInfo(data string) map[string]string {
	memInfo := map[string]string{}

	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			memInfo[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	return memInfo
}
