package environments

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== ExecResult type =====

func TestExecResult(t *testing.T) {
	t.Parallel()

	r := ExecResult{
		Stdout:   "output",
		Stderr:   "error",
		ExitCode: 1,
	}
	assert.Equal(t, "output", r.Stdout)
	assert.Equal(t, "error", r.Stderr)
	assert.Equal(t, 1, r.ExitCode)
}

// ===== LocalEnvironment =====

func TestLocalEnvironment_Name(t *testing.T) {
	t.Parallel()

	env := &LocalEnvironment{}
	assert.Equal(t, "local", env.Name())
}

func TestLocalEnvironment_Execute(t *testing.T) {
	t.Parallel()

	env := &LocalEnvironment{}
	result, err := env.Execute(context.Background(), "echo", []string{"hello"})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "hello")
}

func TestLocalEnvironment_Execute_WithWorkDir(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	env := &LocalEnvironment{WorkDir: tmpDir}
	result, err := env.Execute(context.Background(), "pwd", nil)
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, filepath.Base(tmpDir))
}

func TestLocalEnvironment_Execute_NonZeroExit(t *testing.T) {
	t.Parallel()

	env := &LocalEnvironment{}
	result, err := env.Execute(context.Background(), "sh", []string{"-c", "exit 42"})
	require.NoError(t, err) // non-zero exit is captured in ExitCode, not error
	assert.Equal(t, 42, result.ExitCode)
}

func TestLocalEnvironment_Execute_InvalidCommand(t *testing.T) {
	t.Parallel()

	env := &LocalEnvironment{}
	_, err := env.Execute(context.Background(), "nonexistent_command_xyz", nil)
	assert.Error(t, err)
}

func TestLocalEnvironment_ReadFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(path, []byte("file content"), 0644)
	require.NoError(t, err)

	env := &LocalEnvironment{}
	data, err := env.ReadFile(context.Background(), path)
	require.NoError(t, err)
	assert.Equal(t, "file content", string(data))
}

func TestLocalEnvironment_ReadFile_NotFound(t *testing.T) {
	t.Parallel()

	env := &LocalEnvironment{}
	_, err := env.ReadFile(context.Background(), "/nonexistent/path/file.txt")
	assert.Error(t, err)
}

func TestLocalEnvironment_WriteFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "output.txt")

	env := &LocalEnvironment{}
	err := env.WriteFile(context.Background(), path, []byte("written content"))
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "written content", string(data))
}

func TestLocalEnvironment_Cleanup(t *testing.T) {
	t.Parallel()

	env := &LocalEnvironment{}
	err := env.Cleanup(context.Background())
	assert.NoError(t, err)
}

func TestLocalEnvironment_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ ExecEnvironment = &LocalEnvironment{}
}

// ===== DockerEnvironment =====

func TestDockerEnvironment_Name(t *testing.T) {
	t.Parallel()

	env := &DockerEnvironment{ContainerID: "abc123", Image: "ubuntu"}
	assert.Equal(t, "docker", env.Name())
}

func TestDockerEnvironment_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ ExecEnvironment = &DockerEnvironment{}
}

// ===== SSHEnvironment =====

func TestSSHEnvironment_Name(t *testing.T) {
	t.Parallel()

	env := &SSHEnvironment{Host: "example.com"}
	assert.Equal(t, "ssh", env.Name())
}

func TestSSHEnvironment_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ ExecEnvironment = &SSHEnvironment{}
}

func TestSSHEnvironment_SSHArgs_Basic(t *testing.T) {
	t.Parallel()

	env := &SSHEnvironment{Host: "example.com"}
	args := env.sshArgs()
	assert.Contains(t, args, "example.com")
	assert.Contains(t, args, "-o")
	assert.Contains(t, args, "StrictHostKeyChecking=no")
}

func TestSSHEnvironment_SSHArgs_WithUser(t *testing.T) {
	t.Parallel()

	env := &SSHEnvironment{Host: "example.com", User: "admin"}
	args := env.sshArgs()
	assert.Contains(t, args, "admin@example.com")
}

func TestSSHEnvironment_SSHArgs_WithKeyFile(t *testing.T) {
	t.Parallel()

	env := &SSHEnvironment{Host: "example.com", KeyFile: "/path/to/key"}
	args := env.sshArgs()
	assert.Contains(t, args, "-i")
	assert.Contains(t, args, "/path/to/key")
}

func TestSSHEnvironment_SSHArgs_WithPort(t *testing.T) {
	t.Parallel()

	env := &SSHEnvironment{Host: "example.com", Port: "2222"}
	args := env.sshArgs()
	assert.Contains(t, args, "-p")
	assert.Contains(t, args, "2222")
}

func TestSSHEnvironment_SSHArgs_AllOptions(t *testing.T) {
	t.Parallel()

	env := &SSHEnvironment{
		Host:    "example.com",
		User:    "admin",
		KeyFile: "/key",
		Port:    "2222",
	}
	args := env.sshArgs()
	assert.Contains(t, args, "-i")
	assert.Contains(t, args, "/key")
	assert.Contains(t, args, "-p")
	assert.Contains(t, args, "2222")
	assert.Contains(t, args, "admin@example.com")
}

func TestSSHEnvironment_Cleanup(t *testing.T) {
	t.Parallel()

	env := &SSHEnvironment{Host: "example.com"}
	err := env.Cleanup(context.Background())
	assert.NoError(t, err)
}

// ===== ModalEnvironment =====

func TestModalEnvironment_Name(t *testing.T) {
	t.Parallel()

	env := &ModalEnvironment{AppName: "test-app"}
	assert.Equal(t, "modal", env.Name())
}

func TestModalEnvironment_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ ExecEnvironment = &ModalEnvironment{}
}

func TestModalEnvironment_Cleanup(t *testing.T) {
	t.Parallel()

	env := &ModalEnvironment{}
	err := env.Cleanup(context.Background())
	assert.NoError(t, err)
}

// ===== DaytonaEnvironment =====

func TestDaytonaEnvironment_Name(t *testing.T) {
	t.Parallel()

	env := &DaytonaEnvironment{WorkspaceID: "ws-123"}
	assert.Equal(t, "daytona", env.Name())
}

func TestDaytonaEnvironment_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ ExecEnvironment = &DaytonaEnvironment{}
}

func TestDaytonaEnvironment_Cleanup(t *testing.T) {
	t.Parallel()

	env := &DaytonaEnvironment{}
	err := env.Cleanup(context.Background())
	assert.NoError(t, err)
}

// ===== SingularityEnvironment =====

func TestSingularityEnvironment_Name(t *testing.T) {
	t.Parallel()

	env := &SingularityEnvironment{ImagePath: "/path/to/image.sif"}
	assert.Equal(t, "singularity", env.Name())
}

func TestSingularityEnvironment_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ ExecEnvironment = &SingularityEnvironment{}
}

func TestSingularityEnvironment_Cleanup(t *testing.T) {
	t.Parallel()

	env := &SingularityEnvironment{}
	err := env.Cleanup(context.Background())
	assert.NoError(t, err)
}

// ===== LocalEnvironment execute with stderr =====

func TestLocalEnvironment_Execute_WithStderr(t *testing.T) {
	t.Parallel()

	env := &LocalEnvironment{}
	result, err := env.Execute(context.Background(), "sh", []string{"-c", "echo error >&2"})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stderr, "error")
}

func TestLocalEnvironment_Execute_ContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	env := &LocalEnvironment{}
	_, err := env.Execute(ctx, "sleep", []string{"10"})
	assert.Error(t, err)
}
