package config

// DefaultProvider represents a set of default values.
// Prefer keeping defaults over providing default values local to other
// providers as it guarantees presence of the default values indiffirent to
// the provider set that have been activated.
type DefaultProvider struct {
	weight   int
	registry map[string]Value
	ready    chan struct{}
}

var _ Provider = (*DefaultProvider)(nil)

// NewDefaultProvider is a constructor for DefaultProvider.
func NewDefaultProvider(repo *Repository, weight int) (*DefaultProvider, error) {
	return NewDefaultProviderWithDefaults(repo, weight, map[string]Value{})
}

// NewDefaultProviderWithDefaults is an alternative constructor for
// DefaultProvider. Accepts an extra registry argument as a complete replacement
// for the default one.
func NewDefaultProviderWithDefaults(repo *Repository, weight int, registry map[string]Value) (*DefaultProvider, error) {
	prov := &DefaultProvider{
		weight:   weight,
		registry: registry,
		ready:    make(chan struct{}),
	}
	repo.RegisterProvider(prov)
	return prov, nil
}

// Name returns provider name: default
func (dp *DefaultProvider) Name() string { return "default" }

// Depends returns the list of provider dependencies: none
func (dp *DefaultProvider) Depends() []string { return []string{} }

// Weight returns the provider weight
func (dp *DefaultProvider) Weight() int { return dp.weight }

// SetUp registers all keys from the registry in the repo
func (dp *DefaultProvider) SetUp(repo *Repository) error {
	defer close(dp.ready)
	for k := range dp.registry {
		if err := repo.RegisterKey(NewKey(k), dp); err != nil {
			return err
		}
	}
	return nil
}

// TearDown is a no-op operation for DefaultProvider
func (dp *DefaultProvider) TearDown(*Repository) error { return nil }

// Get is the primary method for fetching values from the default registry
func (dp *DefaultProvider) Get(key Key) (*KeyValue, bool) {
	<-dp.ready
	if val, ok := dp.registry[key.String()]; ok {
		return &KeyValue{Key: key, Value: val}, ok
	}
	return nil, false
}
