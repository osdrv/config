package config

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestDefaultProviderSetUp(t *testing.T) {
	tests := []struct {
		name     string
		registry map[string]Value
		wantRegs []string
	}{
		{
			"empty map",
			map[string]Value{},
			[]string{},
		},
		{
			"defaults",
			defaults,
			[]string{
				"config.path",
				"plugin.path",
				"system.maxprocs",
			},
		},
		{
			"Custom registry",
			map[string]Value{
				"foo.bar.baz": 42,
			},
			[]string{"foo.bar.baz"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			repo := NewRepository()
			prov, err := NewDefaultProviderWithRegistry(repo, 0, testCase.registry)
			if err != nil {
				t.Fatalf("failed to initialize a new default provider: %s", err)
			}
			if err := prov.SetUp(repo); err != nil {
				t.Fatalf("failed to set up default provider: %s", err)
			}

			gotRegs := flattenRepo(repo)
			for _, k := range testCase.wantRegs {
				provs, ok := gotRegs[k]
				if !ok {
					t.Fatalf("failed to find a registration for key %q", k)
				}
				if !reflect.DeepEqual(provs, []Provider{prov}) {
					t.Fatalf("unexpected provider list for key %q: %#v, want: %#v", k, provs, []Provider{prov})
				}
				delete(gotRegs, k)
			}
			if len(gotRegs) > 0 {
				extraKeys := make([]string, 0, len(gotRegs))
				for k := range gotRegs {
					extraKeys = append(extraKeys, k)
				}
				sort.Strings(extraKeys)
				t.Fatalf("unexpected registration keys: %s", strings.Join(extraKeys, ", "))
			}
		})
	}
}
