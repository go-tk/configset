package configset_test

import (
	"errors"
	"testing"

	"github.com/go-tk/configset"
	. "github.com/go-tk/configset"
	"github.com/go-tk/testcase"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfigSet(t *testing.T) {
	type Workspace struct {
		CS ConfigSet
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
			err := w.CS.Load(w.In.MemMapFs, w.In.DirPath, w.In.Environment)
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
			if w.CS.IsLoaded() {
				w.ActSt.JSON = string(w.CS.Dump("", ""))
			}
			assert.Equal(t, w.ExpSt, w.ActSt)
		})
	testcase.RunListParallel(t,
		tc.Copy().
			Given("directory without good configuration files").
			Then("should succeed").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.In.MemMapFs.Mkdir("/my_etc", 0755)
				w.In.DirPath = "/my_etc"
				w.ExpSt.JSON = "{}"
			}),
		tc.Copy().
			Given("directory with configuration files").
			Then("should succeed").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.In.MemMapFs.Mkdir("/my_etc", 0755)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
numbers: [1,2,3]
`), 0644)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: 1.0
author: roy
`), 0644)
				w.In.DirPath = "/my_etc"
				w.ExpSt.JSON = `{"aaa":{"hello":"world","numbers":[1,2,3]},"gogo":{"author":"roy","version":1}}`
			}),
		tc.Copy().
			Given("environment with good overriding values").
			Then("should succeed").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.In.MemMapFs.Mkdir("/my_etc", 0755)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
numbers: [1,2,3]
`), 0644)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: 1.0
author: roy
`), 0644)
				w.In.DirPath = "/my_etc"
				w.In.Environment = []string{
					"FOO=BAR",
					"CONFIGSET.aaa.hello=\"hi\"",
					"CONFIGSET.aaa.numbers.1=-2",
					"CONFIGSET.gogo.version.y=22",
					`CONFIGSET.gogo.version={"x": 1, "y": 2, "z": 3}`,
				}
				w.ExpSt.JSON = `{"aaa":{"hello":"hi","numbers":[1,-2,3]},"gogo":{"author":"roy","version":{"x":1,"y":22,"z":3}}}`
			}),
		tc.Copy().
			Given("directory with bad configuration files").
			Then("should fail").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.In.MemMapFs.Mkdir("/my_etc", 0755)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
numbers: [1,2,3
`), 0644)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: 1.0
author: roy
`), 0644)
				w.In.DirPath = "/my_etc"
				w.ExpOut.ErrStr = "convert yaml to json; filePath=\"/my_etc/aaa.yaml\": yaml: line 3: did not find expected ',' or ']'"
			}),
		tc.Copy().
			Given("environment with bad overriding values (1)").
			Then("should fail").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.In.MemMapFs.Mkdir("/my_etc", 0755)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
numbers: [1,2,3]
`), 0644)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: 1.0
author: roy
`), 0644)
				w.In.DirPath = "/my_etc"
				w.In.Environment = []string{
					"CONFIGSET.aaa.hello='",
				}
				w.ExpOut.ErrStr = "convert yaml to json; key=\"CONFIGSET.aaa.hello\" value=\"'\": yaml: found unexpected end of stream"
			}),
		tc.Copy().
			Given("environment with bad overriding values (2)").
			Then("should fail").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.In.MemMapFs.Mkdir("/my_etc", 0755)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
numbers: [1,2,3]
`), 0644)
				afero.WriteFile(w.In.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: 1.0
author: roy
`), 0644)
				w.In.DirPath = "/my_etc"
				w.In.Environment = []string{
					"CONFIGSET.=1",
				}
				w.ExpOut.ErrStr = "set json value; path=\"\": path cannot be empty"
			}),
	)
}

func TestConfigSet_ReadValue(t *testing.T) {
	type Workspace struct {
		CS   ConfigSet
		Init struct {
			MemMapFs    *afero.MemMapFs
			DirPath     string
			Environment []string
		}
		In struct {
			Path   string
			Config interface{}
		}
		ExpOut, ActOut struct {
			Config interface{}
			ErrStr string
			Err    error
		}
	}
	tc := testcase.New().
		Step(0, func(t *testing.T, w *Workspace) {
			w.Init.MemMapFs = afero.NewMemMapFs().(*afero.MemMapFs)
		}).
		Step(1, func(t *testing.T, w *Workspace) {
			err := w.CS.Load(w.Init.MemMapFs, w.Init.DirPath, w.Init.Environment)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
		}).
		Step(2, func(t *testing.T, w *Workspace) {
			err := w.CS.ReadValue(w.In.Path, w.In.Config)
			if err == nil {
				w.ActOut.Config = w.In.Config
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
				w.Init.MemMapFs.Mkdir("/my_etc", 0755)
				afero.WriteFile(w.Init.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
my_numbers: [1,2,3]
`), 0644)
				afero.WriteFile(w.Init.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: 1.0
author: roy
`), 0644)
				w.Init.DirPath = "/my_etc"
				w.Init.Environment = []string{
					"CONFIGSET.aaa.my_numbers.1=-2",
				}
			}).
			Step(1.5, func(t *testing.T, w *Workspace) {
				type AAA struct {
					Hello     string `json:"hello"`
					MyNumbers []int  `json:"my_numbers"`
				}
				w.In.Path = "aaa"
				w.In.Config = &AAA{}
				w.ExpOut.Config = &AAA{
					Hello:     "world",
					MyNumbers: []int{1, -2, 3},
				}
			}),
		tc.Copy().
			Then("should succeed").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.Init.MemMapFs.Mkdir("/my_etc", 0755)
				afero.WriteFile(w.Init.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
my_numbers: [1,2,3]
`), 0644)
				afero.WriteFile(w.Init.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: [1,2,3]
author: roy
`), 0644)
				w.Init.DirPath = "/my_etc"
			}).
			Step(1.5, func(t *testing.T, w *Workspace) {
				var my_numbers []int
				w.In.Path = "gogo.version"
				w.In.Config = &my_numbers
				w.ExpOut.Config = &[]int{1, 2, 3}
			}),
		tc.Copy().
			Given("unexpected value").
			Then("should fail").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.Init.MemMapFs.Mkdir("/my_etc", 0755)
				afero.WriteFile(w.Init.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
my_numbers: [1,2,3]
`), 0644)
				afero.WriteFile(w.Init.MemMapFs, "/my_etc/gogo.yaml", []byte(`
version: 1
author: 1
`), 0644)
				w.Init.DirPath = "/my_etc"
			}).
			Step(1.5, func(t *testing.T, w *Workspace) {
				type GoGo struct {
					Version int    `json:"version"`
					Author  string `json:"author"`
				}
				w.In.Path = "gogo"
				w.In.Config = &GoGo{}
				w.ExpOut.ErrStr = "unmarshal from json; path=\"gogo\" configType=\"*configset_test.GoGo\": json: cannot unmarshal number into Go struct field GoGo.author of type string"
			}),
		tc.Copy().
			Given("no value corresponding to path").
			Then("should fail").
			Step(.5, func(t *testing.T, w *Workspace) {
				w.Init.MemMapFs.Mkdir("/my_etc", 0755)
				afero.WriteFile(w.Init.MemMapFs, "/my_etc/aaa.yaml", []byte(`
hello: world
my_numbers: [1,2,3]
`), 0644)
				w.Init.DirPath = "/my_etc"
			}).
			Step(1.5, func(t *testing.T, w *Workspace) {
				type GoGo struct {
					Version int    `json:"version"`
					Author  string `json:"author"`
				}
				w.In.Path = "gogo"
				w.In.Config = &GoGo{}
				w.ExpOut.ErrStr = "configset: value not found; path=\"gogo\""
				w.ExpOut.Err = ErrValueNotFound
			}),
	)
}

func TestMustLoad(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.Mkdir("/my_etc", 0755)
	err := afero.WriteFile(fs, "/my_etc/foo.yaml", []byte(`
bar: "
`), 0644)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	fn := *configset.NewFs
	*configset.NewFs = func() afero.Fs { return fs }
	defer func() { *configset.NewFs = fn }()
	assert.PanicsWithValue(t, "load config set: convert yaml to json; filePath=\"/my_etc/foo.yaml\": yaml: line 3: found unexpected end of stream", func() {
		configset.MustLoad("/my_etc")
	})
}

func TestMustReadValue(t *testing.T) {
	fn := *configset.GetEnvironment
	*configset.GetEnvironment = func() []string { return []string{"CONFIGSET.foo.bar=100"} }
	defer func() { *configset.GetEnvironment = fn }()
	assert.PanicsWithValue(t, "read value: unmarshal from json; path=\"foo.bar\" configType=\"*string\": json: cannot unmarshal number into Go value of type string", func() {
		configset.MustLoad("/my_etc")
		var s string
		configset.MustReadValue("foo.bar", &s)
	})
}
