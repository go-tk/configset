package configset_test

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-tk/configset"
)

func Example() {
	// 1. Create configuration files for testing.
	_ = os.Mkdir("./temp", 0755)

	ioutil.WriteFile("./temp/foo.yaml", []byte(`
user_id: 1000
nickname: roy
friends: [dave]
`), 0644)

	ioutil.WriteFile("./temp/bar.yaml", []byte(`
secrets:
  password: s0g00d
  luck_numbers:
    - 1
    - 3
    - 5
`), 0644)

	// 2. Override configuration values with environment variables.
	os.Setenv("CONFIGSET.foo.nickname", "lisa")             // env value should be valid YAML
	os.Setenv("CONFIGSET.foo.friends", "[maria, victoria]") // env value should be valid YAML
	os.Setenv("CONFIGSET.bar.secrets.luck_numbers.1", "99") // env value should be valid YAML

	// 3. Read in configuration files.
	configset.MustLoad("./temp")

	// 4. Dump the configuration set in form of JSON for debugging.
	json := string(configset.Dump("", "  "))
	fmt.Println("===== Dump =====")
	fmt.Print(json)

	// 5. Read a configuration value into a struct.
	var secrets struct {
		Password    string `json:"password"`     // should use json tag rather than yaml tag
		LuckNumbers []int  `json:"luck_numbers"` // should use json tag rather than yaml tag
	}
	configset.MustReadValue("bar.secrets", &secrets)
	fmt.Println("===== ReadValue  =====")
	fmt.Printf("%v\n", secrets)

	// output:
	// ===== Dump =====
	// {
	//   "bar": {
	//     "secrets": {
	//       "luck_numbers": [
	//         1,
	//         99,
	//         5
	//       ],
	//       "password": "s0g00d"
	//     }
	//   },
	//   "foo": {
	//     "friends": [
	//       "maria",
	//       "victoria"
	//     ],
	//     "nickname": "lisa",
	//     "user_id": 1000
	//   }
	// }
	// ===== ReadValue  =====
	// {s0g00d [1 99 5]}
}
