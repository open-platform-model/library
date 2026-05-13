package kernel

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
)

func TestPartial_FlipsConfigPartialFlag(t *testing.T) {
	var cfg validateConfig
	assert.False(t, cfg.partial, "default config has partial=false")

	Partial()(&cfg)
	assert.True(t, cfg.partial, "Partial() option MUST flip cfg.partial to true")
}

func TestPartial_Idempotent(t *testing.T) {
	var cfg validateConfig
	Partial()(&cfg)
	Partial()(&cfg)
	assert.True(t, cfg.partial, "Applying Partial() twice keeps cfg.partial=true")
}

func TestSource_ZeroValueIsAllowed(t *testing.T) {
	var s Source
	assert.False(t, s.Value.Exists(), "zero Source has zero cue.Value")
	assert.Equal(t, "", s.Name)
	assert.Equal(t, "", s.Origin)

	// Confirms the type can be declared and held without panics — the
	// contract is "MUST be loaded with cue.Filename(Origin)" but the
	// library does not enforce non-zero at construction time.
	s.Value = cue.Value{}
	s.Name = "ad-hoc"
	s.Origin = "memory://ad-hoc"
	assert.Equal(t, "ad-hoc", s.Name)
}
