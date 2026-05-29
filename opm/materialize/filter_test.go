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
