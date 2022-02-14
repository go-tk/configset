package configset

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"sigs.k8s.io/yaml"
)

var (
	fsFactory          = func() afero.Fs { return afero.NewOsFs() }
	environmentFactory = func() []string { return os.Environ() }
)

var cs *configSet

// Open sets up the config set.
// All *.yaml files under the given directory will be read into a cache.
// If there are environment variables set such as CONFIGSET.{path}={value},
// the cache will be overwritten according to {paths} and {values}.
func Open(dirPath string) error {
	fs := fsFactory()
	environment := environmentFactory()
	var err error
	cs, err = openConfigSet(fs, dirPath, environment)
	if err != nil {
		return err
	}
	return nil
}

// MustOpen likes Open but panics when an error occurs.
func MustOpen(dirPath string) {
	if err := Open(dirPath); err != nil {
		panic(fmt.Sprintf("open config set: %v", err))
	}
}

// LoadItem finds the value for the given path from the cache and unmarshals
// the given item from that value in form of JSON.
// If no value can be found by the path, ErrValueNotFound is returned.
func LoadItem(path string, item interface{}) error { return cs.LoadItem(path, item) }

// MustLoadItem likes LoadItem but panics when an error occurs.
func MustLoadItem(path string, item interface{}) {
	if err := LoadItem(path, item); err != nil {
		panic(fmt.Sprintf("load item: %v", err))
	}
}

// Dump returns the cache in form of JSON.
func Dump(prefix string, indention string) json.RawMessage { return cs.Dump(prefix, indention) }

type configSet struct {
	raw json.RawMessage
}

func openConfigSet(fs afero.Fs, dirPath string, environment []string) (*configSet, error) {
	rawConfigSet, err := gatherConfigs(fs, dirPath)
	if err != nil {
		return nil, err
	}
	rawConfigSet, err = patchConfigSet(rawConfigSet, environment)
	if err != nil {
		return nil, err
	}
	return &configSet{
		raw: rawConfigSet,
	}, nil
}

func gatherConfigs(fs afero.Fs, dirPath string) (json.RawMessage, error) {
	pattern := filepath.Join(dirPath, "*.yaml")
	filePaths, err := afero.Glob(fs, pattern)
	if err != nil {
		return nil, fmt.Errorf("find files; pattern=%q: %w", pattern, err)
	}
	rawConfigs := make(map[string]json.RawMessage)
	for _, filePath := range filePaths {
		configName := strings.TrimSuffix(filepath.Base(filePath), ".yaml")
		rawConfig, err := afero.ReadFile(fs, filePath)
		if err != nil {
			return nil, fmt.Errorf("read file; filePath=%q: %w", filePath, err)
		}
		rawConfig, err = yaml.YAMLToJSONStrict(rawConfig)
		if err != nil {
			return nil, fmt.Errorf("convert yaml to json; filePath=%q: %w", filePath, err)
		}
		rawConfigs[configName] = rawConfig
	}
	rawConfigSet, err := json.Marshal(rawConfigs)
	if err != nil {
		return nil, fmt.Errorf("marshal to json: %w", err)
	}
	return rawConfigSet, nil
}

func patchConfigSet(rawConfigSet json.RawMessage, environment []string) (json.RawMessage, error) {
	kvs := extractKVs(environment)
	for _, kv := range kvs {
		key, value := kv[0], kv[1]
		data, err := yaml.YAMLToJSONStrict([]byte(value))
		if err != nil {
			return nil, fmt.Errorf("convert yaml to json; key=%q value=%q: %w", key, value, err)
		}
		path := key[len(keyPrefix):]
		rawConfigSet, err = sjson.SetRawBytesOptions(rawConfigSet, path, data, &sjson.Options{
			Optimistic:     true,
			ReplaceInPlace: true,
		})
		if err != nil {
			return nil, fmt.Errorf("set json value; path=%q: %w", path, err)
		}
	}
	return rawConfigSet, nil
}

const keyPrefix = "CONFIGSET."

func extractKVs(environment []string) [][2]string {
	var kvs [][2]string
	for _, rawKV := range environment {
		if !strings.HasPrefix(rawKV, keyPrefix) {
			continue
		}
		i := strings.IndexByte(rawKV, '=')
		if i < 0 {
			continue
		}
		kv := [2]string{rawKV[:i], rawKV[i+1:]}
		kvs = append(kvs, kv)
	}
	return kvs
}

func (cs *configSet) LoadItem(path string, item interface{}) error {
	value := gjson.GetBytes(cs.raw, path).Raw
	if value == "" {
		return fmt.Errorf("%w; path=%q", ErrValueNotFound, path)
	}
	if err := json.Unmarshal([]byte(value), item); err != nil {
		return fmt.Errorf("unmarshal from json; path=%q itemType=\"%T\": %w", path, item, err)
	}
	return nil
}

func (cs *configSet) Dump(prefix string, indention string) json.RawMessage {
	if len(prefix)+len(indention) == 0 {
		raw := make(json.RawMessage, len(cs.raw))
		copy(raw, cs.raw)
		return raw
	}
	var buffer bytes.Buffer
	json.Indent(&buffer, cs.raw, prefix, indention)
	buffer.WriteByte('\n')
	raw := buffer.Bytes()
	return raw
}

// ErrValueNotFound is returned when the JSON value does not exist.
var ErrValueNotFound = errors.New("configset: value not found")
