// Package mcp provides Docker integration for running MCP servers in isolated containers
package mcp

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/gmsas95/myrai-cli/internal/errors"
)

// DockerContainer represents a running Docker container for an MCP server
type DockerContainer struct {
	ID         string
	Name       string
	ServerName string
	Image      string
	Status     ContainerStatus
	StartedAt  time.Time
}

// ContainerStatus represents the status of a container
type ContainerStatus string

const (
	ContainerRunning  ContainerStatus = "running"
	ContainerStopped  ContainerStatus = "stopped"
	ContainerError    ContainerStatus = "error"
	ContainerCreating ContainerStatus = "creating"
)

// DockerConfig holds Docker-specific configuration for MCP servers
type DockerConfig struct {
	Image        string            `json:"image" yaml:"image"`
	Tag          string            `json:"tag" yaml:"tag"`
	Env          map[string]string `json:"env" yaml:"env"`
	Volumes      []VolumeMount     `json:"volumes" yaml:"volumes"`
	Network      string            `json:"network" yaml:"network"`
	CPULimit     string            `json:"cpu_limit" yaml:"cpu_limit"`
	MemoryLimit  string            `json:"memory_limit" yaml:"memory_limit"`
	ReadOnlyFS   bool              `json:"read_only_fs" yaml:"read_only_fs"`
	User         string            `json:"user" yaml:"user"`
	SecurityOpts []string          `json:"security_opts" yaml:"security_opts"`
}

// VolumeMount represents a Docker volume mount
type VolumeMount struct {
	Source   string `json:"source" yaml:"source"`
	Target   string `json:"target" yaml:"target"`
	ReadOnly bool   `json:"read_only" yaml:"read_only"`
}

// DockerManager manages Docker containers for MCP servers
type DockerManager struct {
	containers map[string]*DockerContainer
}

// NewDockerManager creates a new Docker manager
func NewDockerManager() *DockerManager {
	return &DockerManager{
		containers: make(map[string]*DockerContainer),
	}
}

// StartMCPServerInDocker starts an MCP server in a Docker container
func (dm *DockerManager) StartMCPServerInDocker(ctx context.Context, config MCPServerConfig, dockerCfg *DockerConfig) (*DockerContainer, error) {
	// Check if Docker is available
	if err := dm.checkDocker(); err != nil {
		return nil, err
	}

	// Generate container name
	containerName := fmt.Sprintf("myrai-mcp-%s-%d", config.Name, time.Now().Unix())

	// Pull image if needed
	image := dockerCfg.Image
	if dockerCfg.Tag != "" {
		image = fmt.Sprintf("%s:%s", dockerCfg.Image, dockerCfg.Tag)
	}

	if err := dm.pullImage(ctx, image); err != nil {
		return nil, errors.Wrap(err, "DOCKER_001", fmt.Sprintf("failed to pull Docker image: %s", image))
	}

	// Build docker run arguments
	args := []string{
		"run",
		"-d", // Detached mode
		"--name", containerName,
		"--rm", // Remove container when stopped
	}

	// Add resource limits
	if dockerCfg.CPULimit != "" {
		args = append(args, "--cpus", dockerCfg.CPULimit)
	}
	if dockerCfg.MemoryLimit != "" {
		args = append(args, "--memory", dockerCfg.MemoryLimit)
	}

	// Add read-only filesystem option
	if dockerCfg.ReadOnlyFS {
		args = append(args, "--read-only")
	}

	// Add user
	if dockerCfg.User != "" {
		args = append(args, "--user", dockerCfg.User)
	}

	// Add security options
	for _, opt := range dockerCfg.SecurityOpts {
		args = append(args, "--security-opt", opt)
	}

	// Add network
	if dockerCfg.Network != "" {
		args = append(args, "--network", dockerCfg.Network)
	}

	// Add volumes
	for _, vol := range dockerCfg.Volumes {
		mount := fmt.Sprintf("%s:%s", vol.Source, vol.Target)
		if vol.ReadOnly {
			mount = mount + ":ro"
		}
		args = append(args, "-v", mount)
	}

	// Add environment variables
	for k, v := range config.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range dockerCfg.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add image and command
	args = append(args, image)
	if config.Command != "" {
		args = append(args, config.Command)
	}
	args = append(args, config.Args...)

	// Create and start container
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrap(err, "DOCKER_002", fmt.Sprintf("failed to start container: %s", string(output)))
	}

	containerID := strings.TrimSpace(string(output))

	// Wait a moment for container to start
	time.Sleep(500 * time.Millisecond)

	// Check container status
	status, err := dm.getContainerStatus(ctx, containerID)
	if err != nil {
		// Cleanup on failure
		dm.StopContainer(ctx, containerID)
		return nil, errors.Wrap(err, "DOCKER_003", "failed to get container status")
	}

	if status != ContainerRunning {
		logs, _ := dm.GetContainerLogs(ctx, containerID)
		dm.StopContainer(ctx, containerID)
		return nil, errors.New("DOCKER_004", fmt.Sprintf("container failed to start. Logs: %s", logs))
	}

	container := &DockerContainer{
		ID:         containerID,
		Name:       containerName,
		ServerName: config.Name,
		Image:      image,
		Status:     ContainerRunning,
		StartedAt:  time.Now(),
	}

	dm.containers[containerID] = container
	dm.containers[config.Name] = container

	return container, nil
}

// StopContainer stops and removes a Docker container
func (dm *DockerManager) StopContainer(ctx context.Context, containerID string) error {
	cmd := exec.CommandContext(ctx, "docker", "stop", "-t", "5", containerID)
	if err := cmd.Run(); err != nil {
		// Try to kill if stop fails
		exec.CommandContext(ctx, "docker", "kill", containerID).Run()
	}

	// The container will be auto-removed due to --rm flag
	delete(dm.containers, containerID)

	return nil
}

// GetContainer returns a container by ID or name
func (dm *DockerManager) GetContainer(idOrName string) (*DockerContainer, bool) {
	container, ok := dm.containers[idOrName]
	return container, ok
}

// GetContainerLogs retrieves the logs from a container
func (dm *DockerManager) GetContainerLogs(ctx context.Context, containerID string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "logs", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// ListContainers returns all managed containers
func (dm *DockerManager) ListContainers() []*DockerContainer {
	containers := make([]*DockerContainer, 0, len(dm.containers))
	seen := make(map[string]bool)

	for _, c := range dm.containers {
		if !seen[c.ID] {
			containers = append(containers, c)
			seen[c.ID] = true
		}
	}

	return containers
}

// UpdateContainerStatus updates the status of all tracked containers
func (dm *DockerManager) UpdateContainerStatus(ctx context.Context) error {
	for id, container := range dm.containers {
		status, err := dm.getContainerStatus(ctx, id)
		if err != nil {
			container.Status = ContainerError
			continue
		}
		container.Status = status
	}
	return nil
}

// CreateIsolatedNetwork creates a Docker network for isolated MCP servers
func (dm *DockerManager) CreateIsolatedNetwork(ctx context.Context, networkName string) error {
	// Check if network exists
	cmd := exec.CommandContext(ctx, "docker", "network", "inspect", networkName)
	if err := cmd.Run(); err == nil {
		// Network already exists
		return nil
	}

	// Create network
	cmd = exec.CommandContext(ctx, "docker", "network", "create",
		"--driver", "bridge",
		"--internal", // No external connectivity
		networkName,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "DOCKER_005", fmt.Sprintf("failed to create network: %s", string(output)))
	}

	return nil
}

// RemoveIsolatedNetwork removes a Docker network
func (dm *DockerManager) RemoveIsolatedNetwork(ctx context.Context, networkName string) error {
	cmd := exec.CommandContext(ctx, "docker", "network", "rm", networkName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "DOCKER_006", fmt.Sprintf("failed to remove network: %s", string(output)))
	}
	return nil
}

// ============ Private Methods ============

func (dm *DockerManager) checkDocker() error {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return errors.New("DOCKER_007", "Docker is not available. Please install Docker and ensure it's running")
	}
	return nil
}

func (dm *DockerManager) pullImage(ctx context.Context, image string) error {
	// Check if image exists locally
	cmd := exec.CommandContext(ctx, "docker", "images", "-q", image)
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		// Image exists
		return nil
	}

	// Pull image
	cmd = exec.CommandContext(ctx, "docker", "pull", image)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w\n%s", image, err, string(output))
	}

	return nil
}

func (dm *DockerManager) getContainerStatus(ctx context.Context, containerID string) (ContainerStatus, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Status}}", containerID)
	output, err := cmd.Output()
	if err != nil {
		return ContainerError, err
	}

	status := strings.TrimSpace(string(output))
	switch status {
	case "running":
		return ContainerRunning, nil
	case "exited":
		return ContainerStopped, nil
	case "dead":
		return ContainerError, nil
	default:
		return ContainerStatus(status), nil
	}
}

// StartMCPServerInDocker is a convenience function for starting an MCP server in Docker
func StartMCPServerInDocker(config MCPServerConfig, dockerConfig *DockerConfig) (*DockerContainer, error) {
	manager := NewDockerManager()
	return manager.StartMCPServerInDocker(context.Background(), config, dockerConfig)
}

// CreateDefaultDockerConfig creates a secure default Docker configuration
func CreateDefaultDockerConfig(image string) *DockerConfig {
	return &DockerConfig{
		Image:       image,
		Tag:         "latest",
		CPULimit:    "0.5",  // 50% of one CPU
		MemoryLimit: "512m", // 512MB RAM
		ReadOnlyFS:  true,
		User:        "1000:1000", // Non-root user
		SecurityOpts: []string{
			"no-new-privileges:true",
			"seccomp=unconfined",
		},
		Network: "bridge",
		Volumes: []VolumeMount{
			{
				Source:   "/tmp",
				Target:   "/tmp",
				ReadOnly: false,
			},
		},
	}
}

// CreateSecureDockerConfig creates a high-security Docker configuration
func CreateSecureDockerConfig(image string) *DockerConfig {
	cfg := CreateDefaultDockerConfig(image)
	cfg.CPULimit = "0.25" // More restrictive
	cfg.MemoryLimit = "256m"
	cfg.SecurityOpts = append(cfg.SecurityOpts,
		"apparmor=docker-default",
	)
	return cfg
}

// CopyToContainer copies a file into a running container
func (dm *DockerManager) CopyToContainer(ctx context.Context, containerID string, srcPath string, destPath string) error {
	cmd := exec.CommandContext(ctx, "docker", "cp", srcPath, fmt.Sprintf("%s:%s", containerID, destPath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "DOCKER_008", fmt.Sprintf("failed to copy file to container: %s", string(output)))
	}
	return nil
}

// ExecuteInContainer executes a command in a running container
func (dm *DockerManager) ExecuteInContainer(ctx context.Context, containerID string, command []string) (string, error) {
	args := append([]string{"exec", containerID}, command...)
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), errors.Wrap(err, "DOCKER_009", "failed to execute command in container")
	}
	return string(output), nil
}

// ContainerHealthCheck performs a health check on a container
func (dm *DockerManager) ContainerHealthCheck(ctx context.Context, containerID string) error {
	status, err := dm.getContainerStatus(ctx, containerID)
	if err != nil {
		return err
	}

	if status != ContainerRunning {
		return errors.New("DOCKER_010", fmt.Sprintf("container is not running: %s", status))
	}

	return nil
}

// StreamContainerLogs streams container logs to a writer
func (dm *DockerManager) StreamContainerLogs(ctx context.Context, containerID string, follow bool, w io.Writer) error {
	args := []string{"logs"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, containerID)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = w
	cmd.Stderr = w

	return cmd.Run()
}
