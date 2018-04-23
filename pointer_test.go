package jsonpatch2

import "testing"

type ptrTest struct {
	sample string
	target []string
	length int
	valid  bool
}

var ptrTests = []ptrTest{
	{``, []string{}, 0, true},
	{`/`, []string{``}, 1, true},
	{`foo`, []string{}, 0, false},
	{`/foo`, []string{`foo`}, 1, true},
	{`/a~1b`, []string{`a/b`}, 1, true},
	{`/c%d`, []string{`c%d`}, 1, true},
	{`/ `, []string{` `}, 1, true},
	{`/~0`, []string{`~`}, 1, true},
	{`/~`, []string{}, 0, false},
	{`/foo/a~1b/c%d/ /~0`, []string{`foo`, `a/b`, `c%d`, ` `, `~`}, 5, true},
	{`/foo/a~1b/c%d///~0`, []string{`foo`, `a/b`, `c%d`, ``, ``, `~`}, 6, true},
	{`/foo/a~1b/c%d/~//~0`, []string{`foo`, `a/b`, `c%d`, ``, ``, `~`}, 6, false},
	{`foo/a~1b/c%d/~//~0`, []string{`foo`, `a/b`, `c%d`, ``, ``, `~`}, 6, false},
}

func ptrEqual(sample Pointer, target []string) bool {
	if len(sample) != len(target) {
		return false
	}
	for i := range target {
		if target[i] != string(sample[i]) {
			return false
		}
	}
	return true
}

func TestPointers(t *testing.T) {
	for _, test := range ptrTests {
		res, err := NewPointer(test.sample)
		if test.valid {
			if err != nil {
				t.Errorf("`%v` did not create pointer! (%v)", test.sample, err)
			}
		} else {
			if err == nil {
				t.Errorf("`%v` created a pointer when it should not have!", test.sample)
			}
			continue
		}

		if !ptrEqual(res, test.target) {
			t.Errorf("Pointer `%#v` does not match target `%#v`", res, test.target)
		}
		resample := res.String()
		if resample != test.sample {
			t.Errorf("Sample %v cast to %#v, then stringified back to %v", test.sample, res, resample)
		}
		if len(res) != test.length {
			t.Errorf("`%v` created len %v pointer (%#v)", test.sample, len(res), res)
		}
	}
}
