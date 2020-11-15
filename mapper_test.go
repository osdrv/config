package config

import (
	"fmt"
	"reflect"
	"testing"
)

type TestMapper struct {
	conv func(kv *KeyValue) (*KeyValue, error)
}

func NewTestMapper(conv func(kv *KeyValue) (*KeyValue, error)) *TestMapper {
	return &TestMapper{
		conv: conv,
	}
}

func (tm *TestMapper) Map(kv *KeyValue) (*KeyValue, error) {
	return tm.conv(kv)
}

func TestMapperNodeInsert(t *testing.T) {
	mpr := NewTestMapper(func(kv *KeyValue) (*KeyValue, error) {
		return kv, nil
	})
	tests := []struct {
		path string
		exp  *MapperNode
	}{

		{
			"",
			&MapperNode{},
		},
		{
			"foo",
			&MapperNode{
				Children: map[string]*MapperNode{
					"foo": &MapperNode{
						Mpr: mpr,
					},
				},
			},
		},
		{
			"foo.bar",
			&MapperNode{
				Children: map[string]*MapperNode{
					"foo": &MapperNode{
						Children: map[string]*MapperNode{
							"bar": &MapperNode{
								Mpr: mpr,
							},
						},
					},
				},
			},
		},
		{
			"foo.*.bar",
			&MapperNode{
				Children: map[string]*MapperNode{
					"foo": &MapperNode{
						Children: map[string]*MapperNode{
							"*": &MapperNode{
								Children: map[string]*MapperNode{
									"bar": &MapperNode{
										Mpr: mpr,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		t.Run(testCase.path, func(t *testing.T) {
			root := NewMapperNode()
			root.Insert(NewKey(testCase.path), mpr)
			if !reflect.DeepEqual(testCase.exp, root) {
				t.Errorf("Unexpected node structure: want: %#v, got: %#v", testCase.exp, root)
			}
		})
	}
}

func TestMapperNodeFindSingleEntryLookup(t *testing.T) {
	tests := []struct {
		insertPaths []string
		lookupPath  string
	}{
		{
			[]string{"foo", "*"},
			"foo",
		},
		{
			[]string{"foo.bar", "foo.*", "*.bar", "*.*"},
			"foo.bar",
		},
		{
			[]string{"foo.bar.baz", "foo.bar.*", "foo.*.baz", "foo.*.*", "*.bar.baz", "*.bar.*", "*.*.baz", "*.*.*"},
			"foo.bar.baz",
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		for _, insertPath := range testCase.insertPaths {
			t.Run(insertPath, func(t *testing.T) {
				mpr := NewTestMapper(func(kv *KeyValue) (*KeyValue, error) { return kv, nil })
				root := NewMapperNode()
				root.Insert(NewKey(insertPath), mpr)
				v := root.Find(NewKey(testCase.lookupPath))
				if v == nil {
					t.Fatalf("Expected to get a lookup result for key %q, got nil", testCase.lookupPath)
				}
				if v.Mpr != mpr {
					t.Fatalf("Unexpected mapper value returned by the key %q lookup: %#v, want: %#v", testCase.lookupPath, v.Mpr, mpr)
				}
			})
		}
	}
}

func TestMapperNodeFindPrecedence(t *testing.T) {
	convFunc := func(kv *KeyValue) (*KeyValue, error) { return kv, nil }
	mprAstrx, mprExct := NewTestMapper(convFunc), NewTestMapper(convFunc)

	tests := []struct {
		exactPath  string
		astrxPaths []string
	}{
		{
			"foo",
			[]string{"*"},
		},
		{
			"foo.bar",
			[]string{"foo.*", "*.bar", "*.*"},
		},
		{
			"foo.bar.baz",
			[]string{"foo.bar.*", "foo.*.baz", "foo.*.*", "*.bar.baz", "*.bar.*", "*.*.baz", "*.*.*"},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		t.Run(testCase.exactPath, func(t *testing.T) {
			root := NewMapperNode()
			root.Insert(NewKey(testCase.exactPath), mprExct)
			for _, astrxPath := range testCase.astrxPaths {
				root.Insert(NewKey(astrxPath), mprAstrx)
			}
			v := root.Find(NewKey(testCase.exactPath))
			if v == nil {
				t.Fatalf("Expected to get a non-nil lookup result for key %q, git nil", testCase.exactPath)
			}
			if v.Mpr != mprExct {
				t.Fatalf("Unexpected value returned by the key %q lookup: got: %#v, want: %#v", testCase.exactPath, v.Mpr, mprExct)
			}
		})
	}
}

func TestConvMapper(t *testing.T) {
	tests := []struct {
		name      string
		conv      Converter
		expVal    Value
		validIn   []Value
		invalidIn []Value
	}{
		{
			name:      "conversion to Int",
			conv:      ToInt,
			expVal:    42,
			validIn:   []Value{42, "42", intptr(42)},
			invalidIn: []Value{true, "", '0', nil},
		},
		{
			name:      "conversion to Str",
			conv:      ToStr,
			expVal:    "42",
			validIn:   []Value{"42", 42, strptr("42")},
			invalidIn: []Value{intptr(42), nil, false, '0'},
		},
		{
			name:      "conversion to Bool",
			conv:      ToBool,
			expVal:    true,
			validIn:   []Value{true, boolptr(true), "true", "y", 1, "1"},
			invalidIn: []Value{123, "asdf", nil},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			mpr := NewConvMapper(testCase.conv)
			for _, val := range testCase.validIn {
				conv, convErr := mpr.Map(&KeyValue{Key: nil, Value: val})
				if convErr != nil {
					t.Fatalf("Unexpected mapping error for input value %#v", val)
				}
				if !reflect.DeepEqual(conv.Value, testCase.expVal) {
					t.Fatalf("Unexpected mapping value for input value %#v: got: %#v, want: %#v", val, conv.Value, testCase.expVal)
				}
			}
			for _, val := range testCase.invalidIn {
				_, convErr := mpr.Map(&KeyValue{Key: nil, Value: val})
				if convErr == nil {
					t.Fatalf("Expected to get an error while converting %#v, got nil", val)
				}
			}
		})
	}
}

func TestDefineSchema(t *testing.T) {

	conv := func(kv *KeyValue) (*KeyValue, error) {
		return kv, nil
	}

	mpr := NewTestMapper(conv)
	mpr1, mpr2 := NewTestMapper(conv), NewTestMapper(conv)

	tests := []struct {
		name   string
		schema Schema
		want   MapperNode
	}{
		{
			"Nil-schema",
			nil,
			MapperNode{},
		},
		{
			"A mapper",
			NewTestMapper(conv),
			MapperNode{
				Mpr: nil,
			},
		},
		{
			"A converter",
			newTestConverter(convAct{1, true}),
			MapperNode{
				Mpr: nil,
			},
		},
		{
			"A mapper, flat key",
			map[string]Schema{
				"foo": mpr,
			},
			MapperNode{
				Mpr: nil,
				Children: map[string]*MapperNode{
					"foo": &MapperNode{
						Mpr: mpr,
					},
				},
			},
		},
		{
			"Simple __self__",
			map[string]Schema{
				"foo": map[string]Schema{
					"__self__": mpr,
				},
			},
			MapperNode{
				Mpr: nil,
				Children: map[string]*MapperNode{
					"foo": &MapperNode{
						Mpr: mpr,
					},
				},
			},
		},
		{
			"Nested structure",
			map[string]Schema{
				"foo": map[string]Schema{
					"bar": map[string]Schema{
						"baz": mpr1,
					},
				},
				"moo": mpr2,
			},
			MapperNode{
				Children: map[string]*MapperNode{
					"foo": &MapperNode{
						Children: map[string]*MapperNode{
							"bar": &MapperNode{
								Children: map[string]*MapperNode{
									"baz": &MapperNode{
										Mpr: mpr1,
									},
								},
							},
						},
					},
					"moo": &MapperNode{
						Mpr: mpr2,
					},
				},
			},
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			mn := NewMapperNode()
			if err := mn.DefineSchema(testCase.schema); err != nil {
				t.Fatalf("Failed to call DefineSchema(): %s", err)
			}
			if !reflect.DeepEqual(testCase.want, *mn) {
				t.Fatalf("Unexpected value after DefineSchema(): got: %#v, want: %#v", *mn, testCase.want)
			}

		})
	}
}

type fooStruct struct {
	Bar int
}

func TestMap(t *testing.T) {
	convSq := NewTestMapper(func(kv *KeyValue) (*KeyValue, error) {
		v := kv.Value.(int)
		return &KeyValue{Key: kv.Key, Value: v * v}, nil
	})
	fooMpr := NewTestMapper(func(kv *KeyValue) (*KeyValue, error) {
		v := kv.Value.(map[string]Value)
		return &KeyValue{Key: kv.Key, Value: &fooStruct{Bar: v["bar"].(int)}}, nil
	})
	errMpr := NewTestMapper(func(kv *KeyValue) (*KeyValue, error) {
		return nil, fmt.Errorf("This mapper returns an error")
	})
	tests := []struct {
		name    string
		schema  Schema
		inputKV *KeyValue
		wantKV  *KeyValue
		wantErr error
	}{
		{
			"nil-schema",
			nil,
			&KeyValue{Key: NewKey("foo"), Value: 42},
			&KeyValue{Key: NewKey("foo"), Value: 42},
			nil,
		},
		{
			"Simple mapper matching the key",
			map[string]Schema{
				"foo": convSq,
			},
			&KeyValue{Key: NewKey("foo"), Value: 4},
			&KeyValue{Key: NewKey("foo"), Value: 16},
			nil,
		},
		{
			"Simple mapper with unknown key",
			map[string]Schema{
				"foo": convSq,
			},
			&KeyValue{Key: NewKey("bar"), Value: 4},
			&KeyValue{Key: NewKey("bar"), Value: 4},
			nil,
		},
		{
			"Nesting schema definition",
			map[string]Schema{
				"foo": map[string]Schema{
					"__self__": fooMpr,
					"bar":      convSq,
				},
			},
			&KeyValue{Key: NewKey("foo.bar"), Value: 4},
			&KeyValue{Key: NewKey("foo.bar"), Value: 16},
			nil,
		},
		{
			"Composite key lookup",
			map[string]Schema{
				"foo": map[string]Schema{
					"__self__": fooMpr,
					"bar":      convSq,
				},
			},
			&KeyValue{Key: NewKey("foo"), Value: map[string]Value{"bar": 4}},
			&KeyValue{Key: NewKey("foo"), Value: &fooStruct{Bar: 4}},
			nil,
		},
		{
			"Failing mapper",
			map[string]Schema{
				"foo": errMpr,
			},
			&KeyValue{Key: NewKey("foo"), Value: 42},
			nil,
			fmt.Errorf("This mapper returns an error"),
		},
	}

	t.Parallel()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			mn := &MapperNode{}
			if err := mn.DefineSchema(testCase.schema); err != nil {
				t.Fatalf("Failed to call DefineSchema(): %s", err)
			}
			gotKV, gotErr := mn.Map(testCase.inputKV)
			if !reflect.DeepEqual(gotErr, testCase.wantErr) {
				t.Fatalf("Unexpected error on Map() call: got: %s, want: %s", gotErr, testCase.wantErr)
			}
			if testCase.wantKV != nil && !reflect.DeepEqual(gotKV, testCase.wantKV) {
				t.Fatalf("Unexpected value: Map(%#v) = %#v, want: %#v", testCase.inputKV, gotKV, testCase.wantKV)
			}
		})
	}
}
