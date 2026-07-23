package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/DROWNING2003/pi-go/packages/ai/provider"
	"github.com/DROWNING2003/pi-go/packages/coding-agent/rpc"
	"github.com/DROWNING2003/pi-go/packages/storage/config"
)

// runRPCMode starts the JSON-RPC headless loop.
func runRPCMode(stdout, stderr io.Writer, version string) int {
	// Config
	cwd, _ := os.Getwd()
	configDir, _ := os.UserConfigDir()
	configDir = filepath.Join(configDir, "pi-go")
	cfg, _ := config.Load(cwd, configDir)

	// Registry
	reg := provider.NewRegistry(nil)
	provider.RegisterBuiltins(reg)

	// Default model
	modelRef := cfg.Model
	if modelRef == "" {
		modelRef = "deepseek/deepseek-chat"
	}
	m := reg.ResolveModel(modelRef)
	if m == nil {
		fmt.Fprintf(stderr, "unknown model: %s\n", modelRef)
		return 1
	}

	fmt.Fprintf(stderr, "pi-rpc %s (provider=%s, model=%s)\n", version, m.Provider, m.ID)
	if err := rpc.RunRPC(reg, m); err != nil {
		fmt.Fprintf(stderr, "rpc error: %v\n", err)
		return 1
	}
	return 0
}
