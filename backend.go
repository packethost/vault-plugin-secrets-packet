package packet

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/packethost/packngo"
)

const secretType = "packet"

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := NewBackend(conf.System)
	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	return b, nil
}

func NewBackend(system logical.SystemView) *backend {
	var b backend
	b.system = system
	b.Backend = &framework.Backend{
		Help: strings.TrimSpace(backendHelp),

		PathsSpecial: &logical.Paths{
			SealWrapStorage: []string{
				"config",
			},
		},

		Paths: []*framework.Path{
			b.pathListRoles(),
			b.pathRole(),
			b.pathConfig(),
			b.pathCredentials(),
		},

		Secrets: []*framework.Secret{
			b.pathSecrets(),
		},

		BackendType: logical.TypeLogical,
	}
	return &b
}

type backend struct {
	*framework.Backend
	client *packngo.Client
	lock   sync.RWMutex
	system logical.SystemView
}

func (b *backend) Client(ctx context.Context, s logical.Storage) (*packngo.Client, error) {
	b.lock.RLock()

	// If we already have a client, return it
	if b.client != nil {
		b.lock.RUnlock()
		return b.client, nil
	}

	b.lock.RUnlock()

	// Otherwise, attempt to make connection
	entry, err := s.Get(ctx, "config")
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, fmt.Errorf("setup the config first")
	}

	var conf packetSecretsEngineConfig
	if err := entry.DecodeJSON(&conf); err != nil {
		return nil, err
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	// If the client was created during the lock switch, return it
	if b.client != nil {
		return b.client, nil
	}

	httpClient := retryablehttp.NewClient()
	httpClient.Logger = nil

	httpClient.RetryWaitMin = time.Second
	httpClient.RetryWaitMax = 30 * time.Second
	httpClient.RetryMax = 10

	b.client = packngo.NewClientWithAuth("Hashicorp Vault", conf.APIToken, httpClient)
	return b.client, nil
}

// resetClient forces a connection next time Client() is called.
func (b *backend) resetClient(_ context.Context) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.client = nil
}

func (b *backend) invalidate(ctx context.Context, key string) {
	switch key {
	case "config":
		b.resetClient(ctx)
	}
}

const backendHelp = "Packet Secrets Engine for Vault"
