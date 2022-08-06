package configset_test

import (
	"encoding/json"
	"os"
	"testing"

	. "github.com/go-tk/configset"
	"github.com/go-tk/testcase"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestConfigSet_Load(t *testing.T) {
	type C struct {
		fs             *afero.MemMapFs
		dirPath        string
		environment    []string
		expectedJSON   string
		expectedErrStr string
		expectedErr    error
	}
	tc := testcase.New(func(t *testing.T, c *C) {
		t.Parallel()

		var cs ConfigSet
		fs := afero.NewMemMapFs().(*afero.MemMapFs)
		c.fs = fs

		testcase.DoCallback(0, t, c)

		err := cs.Load(fs, c.dirPath, c.environment)
		if c.expectedErrStr != "" {
			assert.EqualError(t, err, c.expectedErrStr)
			if c.expectedErr != nil {
				assert.ErrorIs(t, err, c.expectedErr)
			}
			return
		}
		assert.NoError(t, err)
		json := string(cs.Dump("", ""))
		assert.Equal(t, c.expectedJSON, json)
	})

	var (
		snippet1 = func(t *testing.T, c *C) {
			if err := c.fs.Mkdir("/my_etc/test", 0755); err != nil {
				t.Fatal(err)
			}
			if err := afero.WriteFile(c.fs, "/my_etc/aaa.yaml", []byte(`
hello: world
numbers: [1,2,3]
`), 0644); err != nil {
				t.Fatal(err)
			}
			if err := afero.WriteFile(c.fs, "/my_etc/test.txt", []byte(`
just for fun!
`), 0644); err != nil {
				t.Fatal(err)
			}
			if err := afero.WriteFile(c.fs, "/my_etc/gogo.yaml", []byte(`
version: 1.0
author: roy
`), 0644); err != nil {
				t.Fatal(err)
			}
			c.dirPath = "/my_etc/"
		}
	)

	// directory without configuration files
	tc.Copy().
		SetCallback(0, func(t *testing.T, c *C) {
			c.dirPath = "/"
			c.expectedJSON = "{}"
		}).
		Run(t)

	// directory with configuration files
	tc.Copy().
		SetCallback(0, func(t *testing.T, c *C) {
			snippet1(t, c)
			c.expectedJSON = `{"aaa":{"hello":"world","numbers":[1,2,3]},"gogo":{"author":"roy","version":1}}`
		}).
		Run(t)

	// environment with overriding values
	tc.Copy().
		SetCallback(0, func(t *testing.T, c *C) {
			snippet1(t, c)
			c.environment = []string{
				"FOO=BAR",
				"CONFIGSET.aaa.hello=\"hi\"",
				"CONFIGSET.aaa.numbers.1=-2",
				"CONFIGSET.gogo.version.y=22",
				`CONFIGSET.gogo.version={"x": 1, "y": 2, "z": 3}`,
				"CONFIGSET.gogo",
				"HELLO=WORLD",
			}
			c.expectedJSON = `{"aaa":{"hello":"hi","numbers":[1,-2,3]},"gogo":{"author":"roy","version":{"x":1,"y":22,"z":3}}}`
		}).
		Run(t)

	// environment with bad configuration files
	tc.Copy().
		SetCallback(0, func(t *testing.T, c *C) {
			snippet1(t, c)
			if err := afero.WriteFile(c.fs, "/my_etc/aaa.yaml", []byte(`
hello: world
numbers: [1,2,3
`), 0644); err != nil {
				t.Fatal(err)
			}
			c.expectedErrStr = "convert yaml to json; filePath=\"/my_etc/aaa.yaml\": yaml: line 3: did not find expected ',' or ']'"
		}).
		Run(t)

	// environment with overriding values (1)
	tc.Copy().
		SetCallback(0, func(t *testing.T, c *C) {
			snippet1(t, c)
			c.environment = []string{
				"CONFIGSET.aaa.hello='",
			}
			c.expectedErrStr = "convert yaml to json; key=\"CONFIGSET.aaa.hello\" value=\"'\": yaml: found unexpected end of stream"
		}).
		Run(t)

	// environment with overriding values (2)
	tc.Copy().
		SetCallback(0, func(t *testing.T, c *C) {
			snippet1(t, c)
			c.environment = []string{
				"CONFIGSET.=1",
			}
			c.expectedErrStr = "set json value; path=\"\": path cannot be empty"
		}).
		Run(t)

	// non-existent configuration directory
	tc.Copy().
		SetCallback(0, func(t *testing.T, c *C) {
			c.dirPath = "/helloworld"
			c.expectedErrStr = `read dir; dirPath="/helloworld": open /helloworld: file does not exist`
			c.expectedErr = os.ErrNotExist
		}).
		Run(t)
}

func TestConfigSet_ReadValue(t *testing.T) {
	type C struct {
		path           string
		config         interface{}
		expectedConfig interface{}
		expectedErrStr string
		expectedErr    error
		expectedErrBuf interface{}
	}
	tc := testcase.New(func(t *testing.T, c *C) {
		t.Parallel()

		var cs ConfigSet
		fs := afero.NewMemMapFs().(*afero.MemMapFs)
		if err := fs.Mkdir("/my_etc/test", 0755); err != nil {
			t.Fatal(err)
		}
		if err := afero.WriteFile(fs, "/my_etc/aaa.yaml", []byte(`
hello: world
numbers: [1,2,3]
`), 0644); err != nil {
			t.Fatal(err)
		}
		if err := afero.WriteFile(fs, "/my_etc/gogo.yaml", []byte(`
version: 1.0
author:
  name: roy
  gender: male
`), 0644); err != nil {
			t.Fatal(err)
		}
		err := cs.Load(fs, "/my_etc/", nil)
		if err != nil {
			t.Fatal(err)
		}

		testcase.DoCallback(0, t, c)

		err = cs.ReadValue(c.path, c.config)
		if c.expectedErrStr != "" {
			assert.EqualError(t, err, c.expectedErrStr)
			if c.expectedErr != nil {
				assert.ErrorIs(t, err, c.expectedErr)
			}
			if c.expectedErrBuf != nil {
				assert.ErrorAs(t, err, c.expectedErrBuf)
			}
			return
		}
		assert.NoError(t, err)
		assert.Equal(t, c.expectedConfig, c.config)
	})

	// read 1st level value
	tc.Copy().
		SetCallback(0, func(t *testing.T, c *C) {
			type AAA struct {
				Hello   string `json:"hello"`
				Numbers []int  `json:"numbers"`
			}
			c.path = "aaa"
			c.config = &AAA{}
			c.expectedConfig = &AAA{
				Hello:   "world",
				Numbers: []int{1, 2, 3},
			}
		}).
		Run(t)

	// read 2nd level value (1)
	tc.Copy().
		SetCallback(0, func(t *testing.T, c *C) {
			c.path = "aaa.numbers"
			c.config = &[]int{}
			c.expectedConfig = &[]int{1, 2, 3}
		}).
		Run(t)

	// read 2nd level value (2)
	tc.Copy().
		SetCallback(0, func(t *testing.T, c *C) {
			type Author struct {
				Name   string `json:"name"`
				Gender string `json:"gender"`
			}
			c.path = "gogo.author"
			c.config = &Author{}
			c.expectedConfig = &Author{
				Name:   "roy",
				Gender: "male",
			}
		}).
		Run(t)

	// read non-existent value
	tc.Copy().
		SetCallback(0, func(t *testing.T, c *C) {
			type Author struct {
				Name   string `json:"name"`
				Gender string `json:"gender"`
			}
			c.path = "gogo.author.age"
			c.expectedErrStr = "configset: value not found; path=\"gogo.author.age\""
			c.expectedErr = ErrValueNotFound
		}).
		Run(t)

	// json unmarshal error
	tc.Copy().
		SetCallback(0, func(t *testing.T, c *C) {
			type Author struct {
				Name   string `json:"name"`
				Gender int    `json:"gender"`
			}
			c.path = "gogo.author"
			c.config = &Author{}
			c.expectedErrStr = `unmarshal from json; path="gogo.author" configType="*configset_test.Author": json: cannot unmarshal string into Go struct field Author.gender of type int`
			c.expectedErrBuf = new(*json.UnmarshalTypeError)
		}).
		Run(t)
}
