# Config

## About

A schema-based composable multi-source config library with an intermediate smart
type casting for Golang.

The config providers get the data from static files, environment variables,
command line arguments and others. Every config attribute is computed
independently.

Providers are organized in a hierarchial structure according to pre-defined
weights.

## Config as a tree structure

In a simplest case, a config can be seen as a key-value dictionary.

```go
config := map[string]interface{}{
    "foo": 1,
    "bar": "hello world",
}
```

```
    root
    /  \
  foo  bar
```

This is an example of what we can define a `flat` config: the config object
itself is a root that contains of value leafs.

In practice, massive config objects usually evolve into higher-degree trees.

```go
config := map[string]interface{}{
    "foo": map[string]interface{}{
        "bar": map[string]interface{}{
            "fizz": 3,
            "buzz": 5,
        },
    },
}
```

In this case a config tree looks like:

```
    root
      \
      bar
      / \
  fizz  buzz
```

This library is well suited for storing high-cardinality tree structures.

## Merge trees

The Config library provides a full support for merge trees.
Assume there are 2 config sources, e.g. environmanet variables and a static
config file. Config provides a single entry point for multiple sources by
merging config trees into a single one.

Here is an example:
```go
envVarCfg := map[string]interface{}{
    "foo": map[string]interface{
        "bar": 1,
    },
}

staticFileCfg := map[string]interface{}{
    "foo": map[string]interface{
        "baz": 42,
    },
}
```

In practice, it is not convinient to deal with multiple config sources
independently as this requires a prior established contract on the config key
sourcing (e.g. a convention that MAXPROCS can only come from the environment,
or: additional attributes should be found in command line arguments).

Instead, we prefer to deal with a merge tree: a structure combining all
key-values from all sources so the config user doesn't have to discriminate
between dinstinct config data sources.

What we mean is:

```go
mergeCfg := map[string]interface{}{
    "foo": map[string]interface{}{
        "bar": 1,
        "baz": 42,
    },
}
```

This structure provides an immediate access to all defined key-value pairs.

## Overlapping key resolution

Assume there are 2 data structures:

```go
envCfg := map[string]interface{}{
    "foo": map[string]interface{}{
        "bar": 1,
    },
}

staticFileCfg := map[string]interface{}{
    "foo": map[string]interface{}{
        "bar": 42,
    },
}
```

A merge tree combining these 2 structures is somewhat non-trivial. Given there
is no chance the new structure will return both 1 and 42 for the key `foo.bar`:
there must be a single value. What value should be returned depende on our
preference, which is defined using weights. The value served by a data source
with a highest weight wins. For the sake of simplisity, weights define a global
order of config sources (providers): not per-key.

Having this defined, we can see what would happen if we provide different
weights to the input data structures:

if envCfg has a weight 10 and staticFileCfg is weighted 20, the resulting value
under the key `foo.bar` would be `42` (favoring staticFileCfg). If we flip the
weights, the value returned by a merge structure would be 1. If there is only 1
provider serving this config key, there is no need to resolve the ambiguity.

The merge tree structure is called a `repository`. Config data sources are
called `config providers`.

## Config Providers

A config provider is a module that is responsible for serving config values from
a single source. E.g. env variables, or: docker secrets.

Here is a complete interface a provider implements:

```go
type Provider interface {
	Name() string
	Depends() []string
	SetUp(*Repository) error
	TearDown(*Repository) error
	Get(Key) (*KeyValue, bool)
	Weight() int
}
```

#### Name

All providers must be uniquely identified by a name. The name is used for
initialization dependency resolution.

#### Depends

Providers could be in dependency from another providers. E.g. command line
arguments can specify a location of the static config file. In this case env
variable provider should be initialized and interpreted before the static file
provider is iniitalized: the latter should know the file location config. 

The dependency resolution is a one-time thing: only used to make sure all
pre-requirements are satisfied.

#### SetUp and an Explicit Confing Key Registration

`SetUp` is an initial stage of a provider lifecycle. A bootstrap activity is
expected to be performed at this step. There is one peculiar behavior that is
expected from providers: an explicit config key registration. A provider is
expected to know upfront what config keys it can serve at the `SetUp` point.

A key registration looks like:

```go
func (cp *ConfigProvider) SetUp(repo *Repository) error {
    fooBar := config.NewKey("foo.bar.baz")
    if err := repo.RegisterKey(fooBar, cp); err != nil {
        return err
    }
	return nil
}
```

This method is being caled automaticaly by the repository.

#### TearDown

If a provider initiates a local process runner (e.g. a goroutine checking for
config source updates), calling this method should terminate all background
runners. If a termination is impossible, an error should be returned (e.g. a
termination timeout). This method is being called automatically by the
repository.

#### Get(Key)

This method would be called on a key lookup. There is no obligation for a
provider to return a value for a key even if it registered it (e.g. a config
value could be gone by the moment of invocation if a provider implements a
dynamic config re-build).

#### Weight

A weight is a customer-defined priority of a specific provider on a key
resolution. Can be interpreted as: upon a key resolution, what value should be
returned if several providers can serve it (see `Overlapping key resolution` for
more details).

## Config Repository

A repository is the central acces sobject in the config hierarchy. It is an
umbrella structure for the provider set performing internal key resolution logic
and converting intermediate structure type casting according to the schema.

An interaction with a repo starts with an initalization.

```go
import "github.com/osdrv/config"

cfg := config.NewRepository()
```

An internal config structure is represented by a schema: a special tailored
object instructing the repo on the type conversion.

The entire config structure should be defined in the config schema.

Providers are explicitly registered in the repository:

```go
if _, err := config.NewCliProvider(cfg, 50); err != nil {
    return err
}
if _, err := NewDockerSecretProvider(cfg, 100); err != nil {
    return err
}
```

Note the second argument to provider constructor functions: this is the weight.

## Schema

The Config library is pretty unique: unlike many other libraries, it provides
access to intermediate(aggregate) config object objects. Consider an example:
given a config structure:

```go
cfg := map[string]interface{}{
    "foo": map[string]interface{}{
        "bar": 1,
        "boo": 42,
        "fizz": map[string]interface{}{
            "buzz": "hello!",
        },
    },
}
```

We can represent this structure as a flat key-value dictionary:

```go
cfg := map[string]interface{}{
    "foo.bar": 1,
    "foo.boo": 42,
    "foo.fizz.buzz": "hello!",
}
```

In this case, one would lookup a specific value by a complete key resolution,
like: 

```go
val := cfg["foo.fizz.buzz"]
```

Would lookup by a key "foo" make sense in this case? Well, this really depends
on how flat the inner representation is. From the customer perspective, it makes
a lot of sense. From the inner representation, it introduces some performance
challenges and is rarely implemented by te config libraries. Config excels in it
by promoting the concept of composite keys. It is as simple as it looks in the
original map: a key "foo.bar" indicates a 2-level config key hierarchy.

Now, as we concluded that lookup by key "foo" makes sense, what value whould it
return? Well, the simplest answer is a `map[string]interface{}`. On the other
hand, this representation is a leakage of the config internal implementation. We
return it because we have no straightforward alternative as we don't know
upfront what the customer would prefer.

And this is the problem the schema definition solves: apart from leaf-level type
conversion, it also handles intermediate key type conersion.

Consider the case:

```go
type Foo struct {
    Bar string
    Boo int
}

config := map[string]interface{}{
    "foo": map[string]interace{}{
        "bar": "hello!",
        "boo": 42,
    },
}
```

The schema goal is to instruct the repo on what data type should be used when
all of the listed key lookups happen:

* `foo.bar`: should return a `string`
* `foo.boo`: should return an `int`
* `foo`: should return `Foo` or `*Foo`

A schema for this case could look like:

```go
schema := config.Schema(map[string]config.Schema{
    "foo": map[string]config.Schema{
        "bar": config.ToStr,
        "boo": config.ToInt,

        "__self__": &FooConverer{},
    },
})
```

Primitive converters are defined by the config library. The only missing bit is:
we have to implement a `Foo` converter.

What it sould look like is:

```go
type FooConverter struct {}

func (c *FooConverter) Map(kv *config.KeyValue) (*config.KeyValue, error) {
    var foo Foo
    vmap := kv.Value.(map[string]config.Value)
    if bar, ok := vmap["bar"]; ok {
        foo.Bar = bar.(string)
    }
    if boo, ok := vmap["boo"]; ok {
        foo.Boo = boo.(int)
    }
    return &config.KeyValue{
        Key: kv.Key,
        Value: foo,
    }, nil
}
```

Internal object type conversions are safe as Config will recursively walk the
config tree and perform the conversion bottom-up. Our job here is to gather all
automatically converted structures into a composite data structure.

## Putting it all together

We've touched a few important points of how Config library works. It is time to
see how it works together.

Firstly, distinct providers can contribute to different parts of the config
tree. Config hides the details of what provider served the config by providing
us a unified access interface:

```go
cfg := config.NewRepository()
cfg.DefineSchema(schema)
config.NewCliProvider(cfg, 10)
config.NewDockerSecretProvider(cfg, 20)

//...

if val, ok := cfg.Get(config.NewKey("foo")); ok {
    //...
}
```

In this case, we as Config users are completely abstracted from the fact that
"foo.bar" could be provided by the CLI provider, whereas "foo.boo" might be
coming from the docker secrets provider (or even both, the latter has a higher
priority and will override a value defined using a CLI arg directive). No matter
what specific part of the config object every provider contributed to, a "foo"
key lookup will guarantee to return a `Foo` struct with all values fulfilled
according to the weights defined.

There are 4 built-in providers:

* Defaults: serves a static map of config values that should be returned if no
  other providers returned a value. The provider should have the least weight.
* Environment variables: provides an access to conventional environment
  variables preffixed with a given string. A naming convention used by this
  provider: names are converted to lowercase, an underscore is interpreted as a
  period (key separator), a double underscore is interpreted as a singular
  underscore. Example: `CONFIG_FOO_BAR=hello`.
* Command line arguments: options are supposed to be provided with `-o` key,
  like: `-o foo.bar=hello`
* A yaml config file. This is an example of a provider that declares a
  dependency on cli and env providers before it can safely initialized. The path
  to the file is read from a config value: `config.path`. A program using this
  config provider can be initialized as: `my_bin -o
  config.path=/path/to/config.yaml`, or:
  `CONFIG_CONFIG_PATH=/path/to/config.yaml my_bin`
