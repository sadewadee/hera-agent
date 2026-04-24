// Package memory — bootstrap.go registers all 8 announced memory provider
// plugins into a plugins.Registry so the agent can select one via config.
//
// These providers were announced in RELEASE_v0.2.0 and v0.4.0 but were
// never wired at startup. This file is the missing ~20-line bootstrap.
package memory

import (
	"github.com/sadewadee/hera/plugins"
	"github.com/sadewadee/hera/plugins/memory/byterover"
	"github.com/sadewadee/hera/plugins/memory/hindsight"
	"github.com/sadewadee/hera/plugins/memory/holographic"
	"github.com/sadewadee/hera/plugins/memory/honcho"
	"github.com/sadewadee/hera/plugins/memory/mem0"
	"github.com/sadewadee/hera/plugins/memory/openviking"
	"github.com/sadewadee/hera/plugins/memory/retaindb"
	"github.com/sadewadee/hera/plugins/memory/supermemory"
)

// RegisterBuiltinProviders registers all 8 built-in memory provider plugins
// into the given registry. Call this at startup before NewFromConfig so that
// any non-sqlite provider named in memory.provider config can be resolved.
//
// Registration is cheap (no network, no DB) — the provider is only
// initialized if selected by the user's config.
func RegisterBuiltinProviders(reg *plugins.Registry) {
	reg.RegisterMemoryProvider(mem0.New())
	reg.RegisterMemoryProvider(hindsight.New())
	reg.RegisterMemoryProvider(holographic.New())
	reg.RegisterMemoryProvider(honcho.New())
	reg.RegisterMemoryProvider(byterover.New())
	reg.RegisterMemoryProvider(openviking.New())
	reg.RegisterMemoryProvider(retaindb.New())
	reg.RegisterMemoryProvider(supermemory.New())
}
