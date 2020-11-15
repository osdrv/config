package config

import (
	"os"
	"strings"
)

// Redefined in tests
var envVars = func() []string {
	return os.Environ()
}

func canonise(key string) string {
	k := strings.Replace(key, "_", ".", -1)
	k = strings.Replace(k, "..", "_", -1)
	return strings.ToLower(k)
}

// EnvProvider reads special FLOW_ preffixed environment variables.
// The contract is:
// * Underscores are being transformed to dots in key part (before the first =).
// * There must be exactly 1 `=` sign.
// * Double underscores are converted to singulars and preserved with no dot-conversion.
type EnvProvider struct {
	weight   int
	registry map[string]Value
	ready    chan struct{}

	prefix string
}

var _ Provider = (*EnvProvider)(nil)

func NewEnvProvider(repo *Repository, weight int) (*EnvProvider, error) {
	return NewEnvProviderWithPrefix(repo, weight, "CONFIG_")
}

// NewEnvProvider returns a new instance of EnvProvider.
func NewEnvProviderWithPrefix(repo *Repository, weight int, prefix string) (*EnvProvider, error) {
	prov := &EnvProvider{
		weight: weight,
		ready:  make(chan struct{}),
		prefix: prefix,
	}
	repo.RegisterProvider(prov)

	return prov, nil
}

// Name returns provider name: env
func (ep *EnvProvider) Name() string { return "env" }

// Depends returns provider dependencies: default
func (ep *EnvProvider) Depends() []string { return []string{"default"} }

// Weight returns provider weight
func (ep *EnvProvider) Weight() int { return ep.weight }

// SetUp takes the list of env vars and canonizes them before registration in
// repo. Env vars are expected to be in form FLOW_<K>=<v>. FLOW_ preffix
// would be cleared out.
func (ep *EnvProvider) SetUp(repo *Repository) error {
	defer close(ep.ready)
	registry := make(map[string]Value)
	var k string
	var v interface{}

	for _, kv := range envVars() {
		if !strings.HasPrefix(kv, ep.prefix) {
			continue
		}
		kv = kv[len(ep.prefix):]
		if ix := strings.Index(kv, "="); ix != -1 {
			k, v = kv[:ix], kv[ix+1:]
		} else {
			k, v = kv, true
		}
		k = canonise(k)
		registry[k] = v
		if repo != nil {
			if err := repo.RegisterKey(NewKey(k), ep); err != nil {
				return err
			}
		}
	}

	ep.registry = registry

	return nil
}

// TearDown is a no-op operation for CliProvider
func (ep *EnvProvider) TearDown(_ *Repository) error { return nil }

// Get is the primary method to fetch values from the provider registry.
func (ep *EnvProvider) Get(key Key) (*KeyValue, bool) {
	<-ep.ready
	if val, ok := ep.registry[key.String()]; ok {
		return &KeyValue{Key: key, Value: val}, ok
	}
	return nil, false
}
