package semver

import "fmt"

func ExampleVersion() {
	newer, err := Version("2.0").IsNewerThan("1.0")
	fmt.Printf("2.0 is Newer than 1.0: %t, %s", newer, err)
	// Output:
	// 2.0 is Newer than 1.0: true, %!s(<nil>)
}
