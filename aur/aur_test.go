package aur

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestBatch(t *testing.T) {
	// Note that namePrexixLen is 7 and these are 3 bytes long, for a
	// total of 10 bytes per entry.
	allNames := []string{"abc", "def", "ghi"}

	type testCase struct {
		maxSize  int
		expected [][]string
	}
	cases := []testCase{
		testCase{
			10,
			[][]string{
				[]string{"abc"},
				[]string{"def"},
				[]string{"ghi"},
			},
		},
		testCase{
			11,
			[][]string{
				[]string{"abc"},
				[]string{"def"},
				[]string{"ghi"},
			},
		},
		testCase{
			20,
			[][]string{
				[]string{"abc", "def"},
				[]string{"ghi"},
			},
		},
		testCase{
			29,
			[][]string{
				[]string{"abc", "def"},
				[]string{"ghi"},
			},
		},
		testCase{
			30,
			[][]string{
				[]string{"abc", "def", "ghi"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("maxSize=%d", tc.maxSize),
			func(t *testing.T) {
				equals(t, tc.expected, escapeAndBatch(tc.maxSize, allNames))
			})
	}
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
