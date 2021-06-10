package resource

import (
	"context"

	v1 "k8s.io/api/rbac/v1"

	"github.com/datawire/ambassador/pkg/kates"
	"github.com/telepresenceio/telepresence/v2/pkg/install"
)

type tmClusterRoleBinding int

var TrafficManagerClusterRoleBinding Instance = tmClusterRoleBinding(0)

func (ri tmClusterRoleBinding) roleBinding() *kates.ClusterRoleBinding {
	cr := new(kates.ClusterRoleBinding)
	cr.TypeMeta = kates.TypeMeta{
		Kind:       "ClusterRoleBinding",
		APIVersion: "rbac.authorization.k8s.io/v1",
	}
	cr.ObjectMeta = kates.ObjectMeta{
		Name: install.ManagerAppName,
	}
	return cr
}

func (ri tmClusterRoleBinding) Create(ctx context.Context) error {
	clb := ri.roleBinding()
	clb.Subjects = []v1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "traffic-manager",
			Namespace: "ambassador",
		},
	}
	clb.RoleRef = v1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     install.ManagerAppName,
	}
	return create(ctx, clb)
}

func (ri tmClusterRoleBinding) Exists(ctx context.Context) (bool, error) {
	return exists(ctx, ri.roleBinding())
}

func (ri tmClusterRoleBinding) Delete(ctx context.Context) error {
	return remove(ctx, ri.roleBinding())
}

func (ri tmClusterRoleBinding) Update(_ context.Context) error {
	// Noop
	return nil
}
