package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-platform-model/library/pkg/core"
)

func TestIdentity_String(t *testing.T) {
	cases := []struct {
		name string
		id   core.Identity
		want string
	}{
		{
			name: "k8s namespaced",
			id:   core.Identity{Type: "Deployment", Name: "web", Scope: "default", Group: "apps", Version: "v1"},
			want: "apps/Deployment/default/web",
		},
		{
			name: "k8s cluster-scoped",
			id:   core.Identity{Type: "ClusterRole", Name: "admin", Group: "rbac.authorization.k8s.io", Version: "v1"},
			want: "rbac.authorization.k8s.io/ClusterRole/admin",
		},
		{
			name: "k8s core group",
			id:   core.Identity{Type: "Service", Name: "web", Scope: "default", Version: "v1"},
			want: "Service/default/web",
		},
		{
			name: "compose service",
			id:   core.Identity{Type: "service", Name: "web", Scope: "myproject"},
			want: "service/myproject/web",
		},
		{
			name: "terraform resource",
			id:   core.Identity{Type: "aws_instance", Name: "web", Group: "aws"},
			want: "aws/aws_instance/web",
		},
		{
			name: "bare type-name",
			id:   core.Identity{Type: "job", Name: "etl"},
			want: "job/etl",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.id.String())
		})
	}
}
