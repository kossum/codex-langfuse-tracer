package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/kirilligum/codex-langfuse-tracer/internal/buildinfo"
)

type LangfuseConfig struct {
	Host      string
	PublicKey string
	SecretKey string
}

func CodexHome() string {
	if value := os.Getenv("CODEX_HOME"); value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".codex"
	}
	return filepath.Join(home, ".codex")
}

func DefaultConfigPath() string {
	return filepath.Join(CodexHome(), "config.toml")
}

func DefaultStatePath() string {
	return filepath.Join(CodexHome(), buildinfo.DefaultStateFileName)
}

type langfuseTOML struct {
	MCPServers map[string]mcpServer `toml:"mcp_servers"`
}

type mcpServer struct {
	Env map[string]string `toml:"env"`
}

func Load(path string) (LangfuseConfig, error) {
	// Prefer explicit environment variables (set by the container entrypoint
	// from docker-compose environment:). This keeps the Langfuse credentials
	// OUT of ~/.codex/config.toml, which Codex itself validates as an MCP
	// server table and rejects when the [mcp_servers.langfuse] block has no
	// transport ("invalid transport"). In Docker the env vars are the single
	// source of truth; the config.toml block below is the fallback for
	// standalone (non-Docker) installs that keep the upstream layout.
	host := strings.TrimRight(os.Getenv("LANGFUSE_HOST"), "/")
	publicKey := os.Getenv("LANGFUSE_PUBLIC_KEY")
	secretKey := os.Getenv("LANGFUSE_SECRET_KEY")
	if host != "" && publicKey != "" && secretKey != "" {
		return LangfuseConfig{Host: host, PublicKey: publicKey, SecretKey: secretKey}, nil
	}

	// Fallback: read the [mcp_servers.langfuse.env] block in config.toml.
	var parsed langfuseTOML
	if _, err := toml.DecodeFile(path, &parsed); err != nil {
		return LangfuseConfig{}, fmt.Errorf("missing Langfuse host/public key/secret key in [mcp_servers.langfuse.env] in %s: %w", path, err)
	}

	env := map[string]string(nil)
	if parsed.MCPServers != nil {
		env = parsed.MCPServers["langfuse"].Env
	}
	host = strings.TrimRight(env["LANGFUSE_HOST"], "/")
	publicKey = env["LANGFUSE_PUBLIC_KEY"]
	secretKey = env["LANGFUSE_SECRET_KEY"]
	missing := make([]string, 0, 3)
	if host == "" {
		missing = append(missing, "host")
	}
	if publicKey == "" {
		missing = append(missing, "public key")
	}
	if secretKey == "" {
		missing = append(missing, "secret key")
	}
	if len(missing) > 0 {
		return LangfuseConfig{}, fmt.Errorf("missing Langfuse %s in [mcp_servers.langfuse.env] in %s", strings.Join(missing, "/"), path)
	}

	return LangfuseConfig{Host: host, PublicKey: publicKey, SecretKey: secretKey}, nil
}
