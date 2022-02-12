package configstore

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

var cs *configStore

// Open sets up the config store.
// All *.yaml files under the given directory will be read in and cached in
// memory in form of JSON.
// If there are environment variables set such as CONFIGSTORE.{path}={value},
// the cache will be overwritten according to paths and values.
func Open(dirPath string) error {
	fs := fsFactory()
	environment := environmentFactory()
	var err error
	cs, err = openConfigStore(fs, dirPath, environment)
	if err != nil {
		return err
	}
	return nil
}

// MustOpen likes Open but panics when an error occurs.
func MustOpen(dirPath string) {
	if err := Open(dirPath); err != nil {
		panic(fmt.Sprintf("open config store: %v", err))
	}
}

// Dump returns the JSON that represents the content of the *.yaml files read in.
func Dump() string { return cs.Dump() }

// LoadItem finds the JSON value for the given path from the cache and unmarshals
// the given item from that JSON value.
// If no JSON value can be found by the path, ErrValueNotFound is returned.
func LoadItem(path string, item interface{}) error { return cs.LoadItem(path, item) }

// MustLoadItem likes LoadItem but panics when an error occurs.
func MustLoadItem(path string, item interface{}) {
	if err := LoadItem(path, item); err != nil {
		panic(fmt.Sprintf("load item: %v", err))
	}
}

type configStore struct {
	rawConfigs json.RawMessage
}

func openConfigStore(fs afero.Fs, dirPath string, environment []string) (*configStore, error) {
	rawConfigs, err := gatherConfigs(fs, dirPath)
	if err != nil {
		return nil, err
	}
	rawConfigs, err = patchConfigs(rawConfigs, environment)
	if err != nil {
		return nil, err
	}
	return &configStore{
		rawConfigs: rawConfigs,
	}, nil
}

func gatherConfigs(fs afero.Fs, dirPath string) (json.RawMessage, error) {
	pattern := filepath.Join(dirPath, "*.yaml")
	filePaths, err := afero.Glob(fs, pattern)
	if err != nil {
		return nil, fmt.Errorf("find files; pattern=%q: %w", pattern, err)
	}
	rawConfigSet := make(map[string]json.RawMessage)
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
		rawConfigSet[configName] = rawConfig
	}
	rawConfigs, err := json.Marshal(rawConfigSet)
	if err != nil {
		return nil, fmt.Errorf("marshal to json: %w", err)
	}
	return rawConfigs, nil
}

func patchConfigs(rawConfigs json.RawMessage, environment []string) (json.RawMessage, error) {
	kvs := extractKVsFromEnvironment(environment)
	for _, kv := range kvs {
		key, value := kv[0], kv[1]
		data, err := yaml.YAMLToJSONStrict([]byte(value))
		if err != nil {
			return nil, fmt.Errorf("convert yaml to json; key=%q value=%q: %w", key, value, err)
		}
		path := key[len(keyPrefix):]
		rawConfigs, err = sjson.SetRawBytesOptions(rawConfigs, path, data, &sjson.Options{
			Optimistic:     true,
			ReplaceInPlace: true,
		})
		if err != nil {
			return nil, fmt.Errorf("set json value; path=%q: %w", path, err)
		}
	}
	return rawConfigs, nil
}

const keyPrefix = "CONFIGSTORE."

func extractKVsFromEnvironment(environment []string) [][2]string {
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

func (cs *configStore) Dump() string {
	var buffer bytes.Buffer
	json.Indent(&buffer, cs.rawConfigs, "", "  ")
	return buffer.String()
}

func (cs *configStore) LoadItem(path string, item interface{}) error {
	value := gjson.GetBytes(cs.rawConfigs, path).Raw
	if value == "" {
		return fmt.Errorf("%w; path=%q", ErrValueNotFound, path)
	}
	if err := json.Unmarshal([]byte(value), item); err != nil {
		return fmt.Errorf("unmarshal from json; path=%q itemType=\"%T\": %w", path, item, err)
	}
	return nil
}

// ErrValueNotFound is returned when the JSON value does not exist.
var ErrValueNotFound = errors.New("configstore: value not found")
