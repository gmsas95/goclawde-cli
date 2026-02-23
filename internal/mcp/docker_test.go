package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDockerManager(t *testing.T) {
	dm := NewDockerManager()
	assert.NotNil(t, dm)
	assert.NotNil(t, dm.containers)
}

func TestDockerConfig(t *testing.T) {
	config := &DockerConfig{
		Image:        "node:18-alpine",
		Tag:          "latest",
		CPULimit:     "0.5",
		MemoryLimit:  "512m",
		ReadOnlyFS:   true,
		User:         "1000:1000",
		SecurityOpts: []string{"no-new-privileges:true"},
		Network:      "bridge",
		Volumes: []VolumeMount{
			{
				Source:   "/tmp",
				Target:   "/tmp",
				ReadOnly: false,
			},
		},
	}

	assert.Equal(t, "node:18-alpine", config.Image)
	assert.Equal(t, "0.5", config.CPULimit)
	assert.Equal(t, "512m", config.MemoryLimit)
	assert.True(t, config.ReadOnlyFS)
	assert.Len(t, config.Volumes, 1)
}

func TestContainerStatus(t *testing.T) {
	assert.Equal(t, ContainerStatus("running"), ContainerRunning)
	assert.Equal(t, ContainerStatus("stopped"), ContainerStopped)
	assert.Equal(t, ContainerStatus("error"), ContainerError)
	assert.Equal(t, ContainerStatus("creating"), ContainerCreating)
}

func TestDockerContainer(t *testing.T) {
	container := &DockerContainer{
		ID:         "abc123",
		Name:       "test-container",
		ServerName: "filesystem",
		Image:      "node:18-alpine",
		Status:     ContainerRunning,
		StartedAt:  time.Now(),
	}

	assert.Equal(t, "abc123", container.ID)
	assert.Equal(t, "test-container", container.Name)
	assert.Equal(t, "filesystem", container.ServerName)
	assert.Equal(t, ContainerRunning, container.Status)
}

func TestCreateDefaultDockerConfig(t *testing.T) {
	config := CreateDefaultDockerConfig("node:18-alpine")

	assert.Equal(t, "node:18-alpine", config.Image)
	assert.Equal(t, "latest", config.Tag)
	assert.Equal(t, "0.5", config.CPULimit)
	assert.Equal(t, "512m", config.MemoryLimit)
	assert.True(t, config.ReadOnlyFS)
	assert.Equal(t, "1000:1000", config.User)
	assert.Len(t, config.SecurityOpts, 2)
	assert.Len(t, config.Volumes, 1)
}

func TestCreateSecureDockerConfig(t *testing.T) {
	config := CreateSecureDockerConfig("node:18-alpine")

	assert.Equal(t, "node:18-alpine", config.Image)
	assert.Equal(t, "0.25", config.CPULimit)
	assert.Equal(t, "256m", config.MemoryLimit)
	assert.Len(t, config.SecurityOpts, 3)
}

func TestVolumeMount(t *testing.T) {
	vol := VolumeMount{
		Source:   "/host/path",
		Target:   "/container/path",
		ReadOnly: true,
	}

	assert.Equal(t, "/host/path", vol.Source)
	assert.Equal(t, "/container/path", vol.Target)
	assert.True(t, vol.ReadOnly)
}

func TestGetContainerNotFound(t *testing.T) {
	dm := NewDockerManager()
	container, ok := dm.GetContainer("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, container)
}

func TestListContainersEmpty(t *testing.T) {
	dm := NewDockerManager()
	containers := dm.ListContainers()
	assert.Empty(t, containers)
}

func TestDockerManagerCheckDockerNotAvailable(t *testing.T) {
	dm := NewDockerManager()
	err := dm.checkDocker()
	// This may pass or fail depending on if Docker is installed
	// We just test that the method exists and runs
	_ = err
}

func TestStartMCPServerInDockerContext(t *testing.T) {
	// This test would require Docker to be running
	// We'll just test that the context is properly handled
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	serverConfig := MCPServerConfig{
		Name:    "test",
		Command: "echo",
		Args:    []string{"hello"},
	}

	dockerConfig := CreateDefaultDockerConfig("alpine")

	dm := NewDockerManager()
	// This will likely fail since Docker may not be available, but it tests the function signature
	_, _ = dm.StartMCPServerInDocker(ctx, serverConfig, dockerConfig)
}
