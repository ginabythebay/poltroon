package tar

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestExtractAll(t *testing.T) {
	// The file tree we will use to create a tar file.  Then we will untar
	// it and see if we get something that matches.
	var tree = []string{
		"foo/",
		"foo/bar/",
		"baz.txt",
		"gus/",
		"gus/bob.txt",
	}
	workarea, err := ioutil.TempDir("", "poltroon_extract_test")
	ok(t, err)
	pristine := path.Join(workarea, "pristine")
	writeToDisk(t, pristine, tree)

	tarFile := path.Join(workarea, "test.tar")
	makeTar(t, pristine, tarFile)

	extracted := path.Join(workarea, "extracted")
	ok(t, os.MkdirAll(extracted, 0755))
	reader, err := os.Open(tarFile)
	ok(t, err)
	defer reader.Close()
	ok(t, ExtractAll(reader, extracted))

	expected := makeExpected(extracted, tree)
	found := readUnTarred(t, extracted)
	equals(t, expected, found)
}

func writeToDisk(t *testing.T, root string, tree []string) {
	for _, name := range tree {
		joined := path.Join(root, name)
		if strings.HasSuffix(name, "/") {
			ok(t, os.MkdirAll(joined, 0755))
		} else {
			base := path.Base(name)
			ok(t, ioutil.WriteFile(joined, []byte(base), 0755))
		}
	}
}

func makeTar(t *testing.T, root string, tarFile string) {
	entries, err := ioutil.ReadDir(root)
	ok(t, err)
	cmd := exec.Command("tar", "cf", tarFile)
	cmd.Dir = root
	// Workaround the fact that we can't use * because there is no shell
	for _, e := range entries {
		cmd.Args = append(cmd.Args, e.Name())
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	ok(t, cmd.Run())
}

func makeExpected(root string, tree []string) map[string]bool {
	expected := map[string]bool{}
	for _, s := range tree {
		expected[path.Join(root, s)] = true
	}
	return expected
}

func readUnTarred(t *testing.T, root string) map[string]bool {
	found := map[string]bool{}
	walker := func(filePath string, info os.FileInfo, err error) error {
		ok(t, err)
		if root == filePath {
			return nil
		}
		found[filePath] = true
		if !info.IsDir() {
			contents, err := ioutil.ReadFile(filePath)
			ok(t, err)
			base := path.Base(filePath)
			equals(t, base, string(contents))
		}
		return nil
	}
	ok(t, filepath.Walk(root, walker))
	return found
}

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
