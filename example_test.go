package configstore_test

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-tk/configstore"
)

func Example() {
	// Create configuration files for testing.
	_ = os.Mkdir("./temp", 0755)

	ioutil.WriteFile("./temp/foo.yaml", []byte(`
user_id: 1000
nickname: roy
`), 0644)

	ioutil.WriteFile("./temp/bar.yaml", []byte(`
secrets:
  password: s0g00d
  luck_numbers:
    - 1
    - 3
    - 5
`), 0644)

	// Override configuration items by environment variables.
	os.Setenv("CONFIGSTORE.foo.nickname", "\"lisa\"")
	os.Setenv("CONFIGSTORE.bar.secrets.luck_numbers.1", "99")

	// Read in configuration files.
	configstore.MustOpen("./temp")

	// Dump the whole configuration in form of JSON for debugging.
	json := configstore.Dump()
	fmt.Println("===== Dump =====")
	fmt.Println(json)

	// Load a configstore item into a struct.
	var secrets struct {
		Password    string `json:"password"`     // Use json tag rather than yaml tag
		LuckNumbers []int  `json:"luck_numbers"` // Use json tag rather than yaml tag
	}
	configstore.MustLoadItem("bar.secrets", &secrets)
	fmt.Println("===== MustLoadItem  =====")
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
	//     "nickname": "lisa",
	//     "user_id": 1000
	//   }
	// }
	// ===== MustLoadItem  =====
	// {s0g00d [1 99 5]}
}
