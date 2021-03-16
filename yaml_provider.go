package config

import (
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

const (
	// CfgPathKey is a string constant used globally to reach up the config
	// file path setting.
	CfgPathKey = "config.path"
)

// Redefined in tests
var readRaw = func(source string) (map[interface{}]interface{}, error) {
	out := make(map[interface{}]interface{})
	data, err := ioutil.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("failed to read yaml config file %q: %s", source, err)
	}
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type YamlProvider struct {
	weight   int
	source   string
	options  *YamlProviderOptions
	registry map[string]Value
	ready    chan struct{}
}

type YamlProviderOptions struct {
	Watch bool
}

var _ Provider = (*YamlProvider)(nil)

func NewYamlProvider(repo *Repository, weight int) (*YamlProvider, error) {
	return NewYamlProviderWithOptions(repo, weight, &YamlProviderOptions{})
}

func NewYamlProviderWithOptions(repo *Repository, weight int, options *YamlProviderOptions) (*YamlProvider, error) {
	return NewYamlProviderFromSource(repo, weight, options, "")
}

func NewYamlProviderFromSource(repo *Repository, weight int, options *YamlProviderOptions, source string) (*YamlProvider, error) {
	prov := &YamlProvider{
		source:   source,
		weight:   weight,
		options:  options,
		registry: make(map[string]Value),
		ready:    make(chan struct{}),
	}
	repo.RegisterProvider(prov)
	return prov, nil
}

func (yp *YamlProvider) Name() string      { return "yaml" }
func (yp *YamlProvider) Depends() []string { return []string{"cli", "env"} }
func (yp *YamlProvider) Weight() int       { return yp.weight }

func (yp *YamlProvider) SetUp(repo *Repository) error {
	defer close(yp.ready)

	if len(yp.source) == 0 {
		source, ok := repo.Get(NewKey(CfgPathKey))
		if !ok {
			return fmt.Errorf("Failed to get yaml config path from repo")
		}
		yp.source = source.(string)
	}

	rawData, err := readRaw(yp.source)
	if err != nil {
		return err
	}
	for k, v := range flatten(rawData) {
		yp.registry[k] = v
		if repo != nil {
			if err := repo.RegisterKey(NewKey(k), yp); err != nil {
				return err
			}
		}
	}

	return nil
}

func flatten(in map[interface{}]interface{}) map[string]Value {
	out := make(map[string]Value)
	for k, v := range in {
		if vmap, ok := v.(map[interface{}]interface{}); ok {
			for sk, sv := range flatten(vmap) {
				out[k.(string)+KeySepCh+sk] = Value(sv)
			}
		} else {
			out[k.(string)] = Value(v)
		}
	}
	return out
}

func (yp *YamlProvider) TearDown(repo *Repository) error {
	return nil
}

func (yp *YamlProvider) Get(key Key) (*KeyValue, bool) {
	<-yp.ready
	if v, ok := yp.registry[key.String()]; ok {
		return &KeyValue{Key: key, Value: v}, ok
	}
	return nil, false
}
