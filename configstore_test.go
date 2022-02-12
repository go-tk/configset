package configstore_test

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/go-tk/configstore"
	. "github.com/go-tk/configstore"
	"github.com/go-tk/testcase"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestOpenConfigStore(t *testing.T) {
	type Workspace struct {
		CS *ConfigStore
		In struct {
			MemMapFs    *afero.MemMapFs
			DirPath     string
			Environment []string
		}
		ExpOut, ActOut struct {
			ErrStr string
			Err    error
		}
		ExpSt, ActSt struct {
			JSON string
		}
	}
	tc := testcase.New().
		Step(0, func(t *testing.T, w *Workspace) {
			w.In.MemMapFs = afero.NewMemMapFs().(*afero.MemMapFs)
		}).
		Step(1, func(t *testing.T, w *Workspace) {
			var err error
			w.CS, err = OpenConfigStore(w.In.MemMapFs, w.In.DirPath, w.In.Environment)
			if err != nil {
				w.ActOut.ErrStr = err.Error()
				w.ActOut.Err = err
			}
		}).
		Step(2, func(t *testing.T, w *Workspace) {
			if w.ExpOut.Err == nil {
				w.ActOut.Err = nil
			} else {
				if errors.Is(w.ActOut.Err, w.ExpOut.Err) {
					w.ActOut.Err = nil
					w.ExpOut.Err = nil
				}
			}
			assert.Equal(t, w.ExpOut, w.ActOut)
		}).
		Step(3, func(t *testing.T, w *Workspace) {
			if w.CS != nil {
				w.ActSt.JSON = w.CS.Dump()
			}
			assert.Equal(t, w.ExpSt, w.ActSt)
		})
	testcase.RunListParallel(t,
		tc.Copy().
			Given("directory without good configuration files").
			Then("should succeed").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.In.MemMapFs.Mkdir("/my_etc", 755)
				w.In.DirPath = "/my_etc"
				w.ExpSt.JSON = "{}"
			}),
		tc.Copy().
			Given("directory with configuration files").
			Then("should succeed").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.In.MemMapFs.Mkdir("/my_etc", 755)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
numbers: [1,2,3]
`), 644)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: 1.0
author: roy
`), 644)
				w.In.DirPath = "/my_etc"
				w.ExpSt.JSON = `{
  "aaa": {
    "hello": "world",
    "numbers": [
      1,
      2,
      3
    ]
  },
  "gogo": {
    "author": "roy",
    "version": 1
  }
}`
			}),
		tc.Copy().
			Given("environment with good overriding values").
			Then("should succeed").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.In.MemMapFs.Mkdir("/my_etc", 755)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
numbers: [1,2,3]
`), 644)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: 1.0
author: roy
`), 644)
				w.In.DirPath = "/my_etc"
				w.In.Environment = []string{
					"FOO=BAR",
					"CONFIGSTORE.aaa.hello=\"hi\"",
					"CONFIGSTORE.aaa.numbers.1=-2",
					`CONFIGSTORE.gogo.version={"x": 1, "y": 2, "z": 3}`,
				}
				w.ExpSt.JSON = `{
  "aaa": {
    "hello": "hi",
    "numbers": [
      1,
      -2,
      3
    ]
  },
  "gogo": {
    "author": "roy",
    "version": {
      "x": 1,
      "y": 2,
      "z": 3
    }
  }
}`
			}),
		tc.Copy().
			Given("directory with bad configuration files").
			Then("should fail").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.In.MemMapFs.Mkdir("/my_etc", 755)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
numbers: [1,2,3
`), 644)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: 1.0
author: roy
`), 644)
				w.In.DirPath = "/my_etc"
				w.ExpOut.ErrStr = "convert yaml to json; filePath=\"/my_etc/aaa.yaml\": yaml: line 3: did not find expected ',' or ']'"
			}),
		tc.Copy().
			Given("environment with bad overriding values (1)").
			Then("should fail").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.In.MemMapFs.Mkdir("/my_etc", 755)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
numbers: [1,2,3]
`), 644)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: 1.0
author: roy
`), 644)
				w.In.DirPath = "/my_etc"
				w.In.Environment = []string{
					"CONFIGSTORE.aaa.hello='",
				}
				w.ExpOut.ErrStr = "convert yaml to json; key=\"CONFIGSTORE.aaa.hello\" value=\"'\": yaml: found unexpected end of stream"
			}),
		tc.Copy().
			Given("environment with bad overriding values (2)").
			Then("should fail").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.In.MemMapFs.Mkdir("/my_etc", 755)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
numbers: [1,2,3]
`), 644)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: 1.0
author: roy
`), 644)
				w.In.DirPath = "/my_etc"
				w.In.Environment = []string{
					"CONFIGSTORE.=1",
				}
				w.ExpOut.ErrStr = "set json value; path=\"\": path cannot be empty"
			}),
	)
}

func TestConfigStore_LoadItem(t *testing.T) {
	type Workspace struct {
		CS   *ConfigStore
		Init struct {
			MemMapFs    *afero.MemMapFs
			DirPath     string
			Environment []string
		}
		In struct {
			Path string
			Item interface{}
		}
		ExpOut, ActOut struct {
			Item   interface{}
			ErrStr string
			Err    error
		}
	}
	tc := testcase.New().
		Step(0, func(t *testing.T, w *Workspace) {
			w.Init.MemMapFs = afero.NewMemMapFs().(*afero.MemMapFs)
		}).
		Step(1, func(t *testing.T, w *Workspace) {
			var err error
			w.CS, err = OpenConfigStore(w.Init.MemMapFs, w.Init.DirPath, w.Init.Environment)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
		}).
		Step(2, func(t *testing.T, w *Workspace) {
			err := w.CS.LoadItem(w.In.Path, w.In.Item)
			if err == nil {
				w.ActOut.Item = w.In.Item
			} else {
				w.ActOut.ErrStr = err.Error()
				w.ActOut.Err = err
			}
		}).
		Step(3, func(t *testing.T, w *Workspace) {
			if w.ExpOut.Err == nil {
				w.ActOut.Err = nil
			} else {
				if errors.Is(w.ActOut.Err, w.ExpOut.Err) {
					w.ActOut.Err = nil
					w.ExpOut.Err = nil
				}
			}
			assert.Equal(t, w.ExpOut, w.ActOut)
		})
	testcase.RunListParallel(t,
		tc.Copy().
			Then("should succeed").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.Init.MemMapFs.Mkdir("/my_etc", 755)
				afero.WriteFile(w.Init.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
my_numbers: [1,2,3]
`), 644)
				afero.WriteFile(w.Init.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: 1.0
author: roy
`), 644)
				w.Init.DirPath = "/my_etc"
				w.Init.Environment = []string{
					"CONFIGSTORE.aaa.my_numbers.1=-2",
				}
			}).
			Step(1.5, func(t *testing.T, w *Workspace) {
				type AAA struct {
					Hello     string `json:"hello"`
					MyNumbers []int  `json:"my_numbers"`
				}
				w.In.Path = "aaa"
				w.In.Item = &AAA{}
				w.ExpOut.Item = &AAA{
					Hello:     "world",
					MyNumbers: []int{1, -2, 3},
				}
			}),
		tc.Copy().
			Then("should succeed").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.Init.MemMapFs.Mkdir("/my_etc", 755)
				afero.WriteFile(w.Init.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
my_numbers: [1,2,3]
`), 644)
				afero.WriteFile(w.Init.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: [1,2,3]
author: roy
`), 644)
				w.Init.DirPath = "/my_etc"
			}).
			Step(1.5, func(t *testing.T, w *Workspace) {
				var my_numbers []int
				w.In.Path = "gogo.version"
				w.In.Item = &my_numbers
				w.ExpOut.Item = &[]int{1, 2, 3}
			}),
		tc.Copy().
			Given("unexpected value").
			Then("should fail").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.Init.MemMapFs.Mkdir("/my_etc", 755)
				afero.WriteFile(w.Init.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
my_numbers: [1,2,3]
`), 644)
				afero.WriteFile(w.Init.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: 1
author: 1
`), 644)
				w.Init.DirPath = "/my_etc"
			}).
			Step(1.5, func(t *testing.T, w *Workspace) {
				type GoGo struct {
					Version int    `json:"version"`
					Author  string `json:"author"`
				}
				w.In.Path = "gogo"
				w.In.Item = &GoGo{}
				w.ExpOut.ErrStr = "unmarshal from json; path=\"gogo\" itemType=\"*configstore_test.GoGo\": json: cannot unmarshal number into Go struct field GoGo.author of type string"
			}),
		tc.Copy().
			Given("no value corresponding to path").
			Then("should fail").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.Init.MemMapFs.Mkdir("/my_etc", 755)
				afero.WriteFile(w.Init.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
my_numbers: [1,2,3]
`), 644)
				w.Init.DirPath = "/my_etc"
			}).
			Step(1.5, func(t *testing.T, w *Workspace) {
				type GoGo struct {
					Version int    `json:"version"`
					Author  string `json:"author"`
				}
				w.In.Path = "gogo"
				w.In.Item = &GoGo{}
				w.ExpOut.ErrStr = "configstore: value not found; path=\"gogo\""
				w.ExpOut.Err = ErrValueNotFound
			}),
	)
}

func TestMustOpen(t *testing.T) {
	_ = os.Mkdir("./temp", 0755)
	err := ioutil.WriteFile("./temp/foo.yaml", []byte(`
bar: "
`), 0644)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.PanicsWithValue(t, "open config store: convert yaml to json; filePath=\"temp/foo.yaml\": yaml: line 3: found unexpected end of stream", func() {
		configstore.MustOpen("./temp")
	})
}

func TestMustLoadItem(t *testing.T) {
	_ = os.Mkdir("./temp", 0755)
	err := ioutil.WriteFile("./temp/foo.yaml", []byte(`
bar: 100
`), 0644)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.PanicsWithValue(t, "load item: unmarshal from json; path=\"foo.bar\" itemType=\"*string\": json: cannot unmarshal number into Go value of type string", func() {
		configstore.MustOpen("./temp")
		var s string
		configstore.MustLoadItem("foo.bar", &s)
	})
}
