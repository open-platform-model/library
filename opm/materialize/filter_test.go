package materialize

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterVersions(t *testing.T) {
	published := []string{"v0.1.0", "v0.1.1", "v0.2.0"}

	tests := []struct {
		name      string
		published []string
		filter    *subscriptionFilter
		want      []string
		wantErr   bool
	}{
		{
			name:      "no filter selects highest",
			published: published,
			filter:    nil,
			want:      []string{"v0.2.0"},
		},
		{
			name:      "empty filter selects highest",
			published: published,
			filter:    &subscriptionFilter{},
			want:      []string{"v0.2.0"},
		},
		{
			name:      "range restricts the set",
			published: published,
			filter:    &subscriptionFilter{Range: ">=0.1.0 <0.2.0"},
			want:      []string{"v0.1.0", "v0.1.1"},
		},
		{
			name:      "deny excludes an in-range version",
			published: published,
			filter:    &subscriptionFilter{Range: ">=0.1.0 <0.2.0", Deny: []string{"0.1.1"}},
			want:      []string{"v0.1.0"},
		},
		{
			name:      "allow includes an out-of-range version",
			published: published,
			filter:    &subscriptionFilter{Range: ">=0.1.0 <0.2.0", Allow: []string{"0.2.0"}},
			want:      []string{"v0.1.0", "v0.1.1", "v0.2.0"},
		},
		{
			name:      "allow normalizes a v-prefixed entry",
			published: published,
			filter:    &subscriptionFilter{Range: "<0.1.1", Allow: []string{"v0.2.0"}},
			want:      []string{"v0.1.0", "v0.2.0"},
		},
		{
			name:      "deny wins over allow on the same version",
			published: published,
			filter:    &subscriptionFilter{Allow: []string{"0.2.0"}, Deny: []string{"0.2.0"}},
			want:      []string{"v0.1.0", "v0.1.1"},
		},
		{
			name:      "no published versions yields nil",
			published: nil,
			filter:    &subscriptionFilter{Range: ">=0.1.0"},
			want:      nil,
		},
		{
			name:      "unparseable range errors",
			published: published,
			filter:    &subscriptionFilter{Range: "not-a-constraint!!"},
			wantErr:   true,
		},
		{
			name:      "unparseable deny errors",
			published: published,
			filter:    &subscriptionFilter{Deny: []string{"garbage"}},
			wantErr:   true,
		},
		{
			name:      "no filter excludes pre-release",
			published: []string{"v0.5.0", "v0.5.1", "v0.6.0-dev.1"},
			filter:    nil,
			want:      []string{"v0.5.1"},
		},
		{
			name:      "no filter falls back to highest pre-release when no stable",
			published: []string{"v0.6.0-dev.1", "v0.6.0-dev.2"},
			filter:    nil,
			want:      []string{"v0.6.0-dev.2"},
		},
		{
			name:      "allow opts a pre-release into a stable range",
			published: []string{"v0.5.0", "v0.5.1", "v0.6.0-dev.1"},
			filter:    &subscriptionFilter{Range: ">=0.5.0 <0.6.0", Allow: []string{"0.6.0-dev.1"}},
			want:      []string{"v0.5.0", "v0.5.1", "v0.6.0-dev.1"},
		},
		{
			name:      "range carrying a pre-release identifier opts pre-releases in",
			published: []string{"v0.5.0", "v0.6.0-dev.1", "v0.6.0"},
			filter:    &subscriptionFilter{Range: ">=0.6.0-dev.0 <0.7.0"},
			want:      []string{"v0.6.0-dev.1", "v0.6.0"},
		},
		{
			// The enhancement 0006 OQ18 shape: an open pre-release range admits
			// the whole -alpha/-dev family, CI dev tags included.
			name:      "pre-release range admits mixed alpha and dev families",
			published: []string{"v1.0.0-alpha", "v1.0.0-alpha.1", "v1.0.0-dev.1784212239.g0c11c12"},
			filter:    &subscriptionFilter{Range: ">=1.0.0-alpha"},
			want:      []string{"v1.0.0-alpha", "v1.0.0-alpha.1", "v1.0.0-dev.1784212239.g0c11c12"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterVersions(tt.published, tt.filter)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHighestStable(t *testing.T) {
	tests := []struct {
		name      string
		published []string
		want      string
	}{
		{
			name:      "all stable returns highest",
			published: []string{"v0.4.0", "v0.5.0", "v0.5.1"},
			want:      "v0.5.1",
		},
		{
			name:      "skips a higher pre-release",
			published: []string{"v0.5.0", "v0.5.1", "v0.6.0-dev.1"},
			want:      "v0.5.1",
		},
		{
			name:      "pre-release-only falls back to highest overall",
			published: []string{"v0.6.0-dev.1", "v0.6.0-dev.2"},
			want:      "v0.6.0-dev.2",
		},
		{
			name:      "unparseable trailing entry is skipped",
			published: []string{"v0.5.1", "not-semver"},
			want:      "v0.5.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, highestStable(tt.published))
		})
	}
}
