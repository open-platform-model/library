// Package cueenv builds the environment slice a cue/load or mod/modconfig
// call consults when the caller wants to override CUE_REGISTRY or
// CUE_CACHE_DIR for that one operation.
//
// The override is a fresh copy of the process environment with the chosen
// variables replaced or appended; the process environment itself is never
// written (no os.Setenv), so a long-running consumer can run overlapping
// loads with different registries. When nothing is overridden the result is
// nil, which both load.Config.Env and modconfig.Config.Env document as "use
// the process environment", so the SDK reads it exactly as it would without
// the kernel in between.
//
// This package is under opm/internal/ so every package under opm/ that
// touches cue/load (the file and registry loaders, the schema loader and the
// render stage) shares one implementation.
package cueenv

import (
	"os"
	"strings"
)

// Override returns nil when both registry and cacheDir are empty, and
// otherwise a copy of os.Environ() in which each non-empty argument replaces
// the existing CUE_REGISTRY / CUE_CACHE_DIR entry or is appended when the
// variable is not set. An empty argument leaves its variable as the process
// has it.
func Override(registry, cacheDir string) []string {
	if registry == "" && cacheDir == "" {
		return nil
	}
	env := os.Environ()
	env = set(env, "CUE_REGISTRY", registry)
	env = set(env, "CUE_CACHE_DIR", cacheDir)
	return env
}

// set replaces the first KEY= entry of env with KEY=value, or appends one
// when no entry exists. An empty value leaves env unchanged.
func set(env []string, key, value string) []string {
	if value == "" {
		return env
	}
	prefix := key + "="
	entry := prefix + value
	for i, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			env[i] = entry
			return env
		}
	}
	return append(env, entry)
}
