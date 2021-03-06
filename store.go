// Package store is a dead simple configuration manager for Go applications.
package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v2"
	"errors"
	"runtime"
)

// MarshalFunc is any marshaler.
type MarshalFunc func(v interface{}) ([]byte, error)

// UnmarshalFunc is any unmarshaler.
type UnmarshalFunc func(data []byte, v interface{}) error

var (
	applicationName = ""
	formats         = map[string]format{}
)

type format struct {
	m  MarshalFunc
	um UnmarshalFunc
}

func init() {
	formats["json"] = format{m: json.Marshal, um: json.Unmarshal}
	formats["yaml"] = format{m: yaml.Marshal, um: yaml.Unmarshal}
	formats["yml"] = format{m: yaml.Marshal, um: yaml.Unmarshal}

	formats["toml"] = format{
		m: func(v interface{}) ([]byte, error) {
			b := bytes.Buffer{}
			err := toml.NewEncoder(&b).Encode(v)
			return b.Bytes(), err
		},
		um: toml.Unmarshal,
	}
}

//// Init sets up a unique application name that will be used for name of the
//// configuration directory on the file system. By default, Store puts all the
//// config data to to $XDG_CONFIG_HOME or $HOME on Linux systems
//// and to %APPDATA% on Windows.
////
//// Beware: Store will panic on any sensitive calls unless you run Init inb4.
//func Init(application string) {
//	applicationName = application
//}

type Store struct {
	path string
}

func formatPath(path string) string {
	if !os.IsPathSeparator(path[0]) {
		path = buildPlatformPath(path)
	}
	if os.IsPathSeparator(path[len(path)-1]){
		return path
	}
	return path+string(os.PathSeparator)
}

func NewStore(path string) *Store {
	return &Store{formatPath(path)}
}

// Register is the way you register configuration formats, by mapping some
// file name extension to corresponding marshal and unmarshal functions.
// Once registered, the format given would be compatible with Load and Save.
func Register(extension string, m MarshalFunc, um UnmarshalFunc) {
	formats[extension] = format{m, um}
}

// Load reads a configuration from `path` and puts it into `v` pointer. Store
// supports either JSON, TOML or YAML and will deduce the file format out of
// the filename (.json/.toml/.yaml). For other formats of custom extensions
// please you LoadWith.
//
// Path is a full filename, including the file extension, e.g. "foobar.json".
// If `path` doesn't exist, Load will create one and emptify `v` pointer by
// replacing it with a newly created object, derived from type of `v`.
//
func (s *Store) Load(name string, v interface{}) error {
	path := s.path + name
	if format, ok := formats[extension(path)]; ok {
		return LoadWith(path, v, format.um)
	}
	return errors.New("store: unknown configuration format")
}

// Save puts a configuration from `v` pointer into a file `path`. Store
// supports either JSON, TOML or YAML and will deduce the file format out of
// the filename (.json/.toml/.yaml). For other formats of custom extensions
// please you LoadWith.
//
// Path is a full filename, including the file extension, e.g. "foobar.json".
//
func (s *Store) Save(name string, v interface{}) error {
	path := s.path + name
	if format, ok := formats[extension(path)]; ok {
		return SaveWith(path, v, format.m)
	}
	return errors.New("store: unknown configuration format")
}

// LoadWith loads the configuration using any unmarshaler at all.
func LoadWith(path string, v interface{}, um UnmarshalFunc) error {
	data, err := ioutil.ReadFile(path)

	if err != nil {
		// There is a chance that file we are looking for
		// just doesn't exist. In this case we are supposed
		// to create an empty configuration file, based on v.
		empty := reflect.New(reflect.TypeOf(v))
		//if innerErr := Save(path, &empty); innerErr != nil {
		//	// Smth going on with the file system... returning error.
		//	return err
		//}

		v = empty

		return nil
	}

	if err := um(data, v); err != nil {
		return fmt.Errorf("store: failed to unmarshal %s: %v", path, err)
	}

	return nil
}

// SaveWith saves the configuration using any marshaler at all.
func SaveWith(path string, v interface{}, m MarshalFunc) error {
	var b bytes.Buffer

	if data, err := m(v); err == nil {
		b.Write(data)
	} else {
		return fmt.Errorf("store: failed to marshal %s: %v", path, err)
	}

	b.WriteRune('\n')

	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}

	if err := ioutil.WriteFile(path, b.Bytes(), os.ModePerm); err != nil {
		return err
	}

	return nil
}

func extension(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i+1:]
		}
	}

	return ""
}

// buildPlatformPath builds a platform-dependent path for relative path given.
func buildPlatformPath(path string) string {
	//if runtime.GOOS == "windows" {
	//	return fmt.Sprintf("%s\\%s", os.Getenv("APPDATA"), path)
	//}

	if runtime.GOOS == "linux" {
		prefix := ""
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			prefix = xdg
		} else {
			prefix = os.Getenv("HOME") + "/.config"
		}
		return fmt.Sprintf("%s/%s", prefix, path)
	}

	return path
}

