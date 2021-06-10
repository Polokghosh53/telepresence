package resource

import (
	"context"

	"github.com/datawire/ambassador/pkg/kates"
	"github.com/telepresenceio/telepresence/v2/pkg/install"
)

type injectorSecret int

var AgentInjectorSecret Instance = injectorSecret(0)

func (ri injectorSecret) secret(ctx context.Context) *kates.Secret {
	sec := new(kates.Secret)
	sec.TypeMeta = kates.TypeMeta{
		Kind:       "Secret",
		APIVersion: "v1",
	}
	sec.ObjectMeta = kates.ObjectMeta{
		Namespace: getScope(ctx).namespace,
		Name:      install.AgentInjectorTLSName,
	}
	return sec
}

func (ri injectorSecret) Create(ctx context.Context) (err error) {
	sc := getScope(ctx)
	if sc.crtPem, sc.keyPem, sc.caPem, err = install.GenerateKeys(); err != nil {
		return err
	}
	sec := ri.secret(ctx)
	sec.Data = map[string][]byte{
		"crt.pem": sc.crtPem,
		"key.pem": sc.keyPem,
		"ca.pem":  sc.caPem,
	}
	return create(ctx, sec)
}

func (ri injectorSecret) Exists(ctx context.Context) (bool, error) {
	return exists(ctx, ri.secret(ctx))
}

func (ri injectorSecret) Delete(ctx context.Context) error {
	return remove(ctx, ri.secret(ctx))
}

func (ri injectorSecret) Update(_ context.Context) error {
	// Noop
	return nil
}
