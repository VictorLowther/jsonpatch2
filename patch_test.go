package jsonpatch2

import (
	"encoding/json"
	"reflect"
	"testing"
)

type opTest struct {
	desc        string
	src         string
	final       string
	patch       string
	pass        bool
	failidx     int
	shouldPatch bool
}

var opTests = []opTest{
	// Basic "test" tests
	{
		`Basic equality test 1`,
		`{"foo":5}`,
		`{"foo":5}`,
		`[{"op":"test","path":"/foo","value":5}]`,
		true,
		0,
		false,
	},
	{
		`Basic equality test 2`,
		`{"foo":5}`,
		`{"foo":5}`,
		`[{"op":"test","path":"/foo","value":6}]`,
		false,
		0,
		false,
	},
	// Whole-document "test" tests
	{
		`Whole document equality test 1`,
		`{"foo":5}`,
		`{"foo":5}`,
		`[{"op":"test","path":"","value":{"foo":5}}]`,
		true,
		0,
		false,
	},
	{
		`Whole document equality test 2`,
		`{"foo":5}`,
		`{"foo":5}`,
		`[{"op":"test","path":"","value":{"foo":6}}]`,
		false,
		0,
		false,
	},

	// Nested object "test"
	{
		`Nested document equality test 1`,
		`{"foo":{"bar":5}}`,
		`{"foo":{"bar":5}}`,
		`[{"op":"test","path":"/foo/bar","value":5}]`,
		true,
		0,
		false,
	},
	{
		`Nested document equality test 2`,
		`{"foo":{"bar":5}}`,
		`{"foo":{"bar":5}}`,
		`[{"op":"test","path":"/foo/bar","value":6}]`,
		false,
		0,
		false,
	},
	{
		`Nested document equality test 3`,
		`{"foo":{"bar":5}}`,
		`{"foo":{"bar":5}}`,
		`[{"op":"test","path":"/foo","value":{"bar":5}}]`,
		true,
		0,
		false,
	},
	{
		`Nested document equality test 4`,
		`{"foo":{"bar":5}}`,
		`{"foo":{"bar":5}}`,
		`[{"op":"test","path":"/foo/bar","value":{"bar":6}}]`,
		false,
		0,
		false,
	},
	{
		`Nested document equality test 6`,
		`{"foo":["bar",5]}`,
		`{"foo":["bar",5]}`,
		`[{"op":"test","path":"/foo","value":["bar",5]}]`,
		true,
		0,
		false,
	},
	{
		`Nested document equality test 7`,
		`{"foo":["bar",5]}`,
		`{"foo":["bar",5]}`,
		`[{"op":"test","path":"/foo","value":["bar",6]}]`,
		false,
		0,
		false,
	},
	// Array indexing "test"
	{
		`Array indexing document equality test 1`,
		`{"foo":["bar",5]}`,
		`{"foo":["bar",5]}`,
		`[{"op":"test","path":"/foo/0","value":"bar"}]`,
		true,
		0,
		false,
	},
	{
		`Array indexing document equality test 1`,
		`{"foo":["bar",5]}`,
		`{"foo":["bar",5]}`,
		`[{"op":"test","path":"/foo/-1","value":5}]`,
		true,
		0,
		false,
	},
	{
		`Array out of bounds index test 1`,
		`{"foo":["bar",5]}`,
		`{"foo":["bar",5]}`,
		`[{"op":"test","path":"/foo/-2","value":5}]`,
		false,
		0,
		false,
	},
	{
		`Array out of bounds index test 2`,
		`{"foo":["bar",5]}`,
		`{"foo":["bar",5]}`,
		`[{"op":"test","path":"/foo/2","value":5}]`,
		false,
		0,
		false,
	},
	// Object adding and removing
	{
		`Basic document add test 1`,
		`{"foo":["bar",5]}`,
		`{"foo":["bar",5],"bar":5}`,
		`[{"op":"add","path":"/bar","value":5}]`,
		true,
		0,
		true,
	},
	{
		`Basic document add test 2`,
		`{"foo":["bar",5],"bar":5}`,
		`{"foo":["bar",5]}`,
		`[{"op":"remove","path":"/bar"}]`,
		true,
		0,
		true,
	},
	{
		`Basic document add test 3`,
		`{"foo":["bar",5]}`,
		`{"foo":["bar",5],"bar":5}`,
		`[{"op":"add","path":"/bar/baz","value":5}]`,
		false,
		0,
		false,
	},
	{
		`Basic document add test 4`,
		`{"foo":["bar",5],"bar":5}`,
		`{"foo":["bar",5]}`,
		`[{"op":"remove","path":"/baz"}]`,
		false,
		0,
		false,
	},

	// Nested object adding and removing
	{
		`Nested document add test 1`,
		`{"foo":{"bar":5}}`,
		`{"foo":{"bar":5,"baz":6}}`,
		`[{"op":"add","path":"/foo/baz","value":6}]`,
		true,
		0,
		true,
	},
	{
		`Nested document add test 2`,
		`{"foo":{"bar":5,"baz":6}}`,
		`{"foo":{"bar":5}}`,
		`[{"op":"remove","path":"/foo/baz"}]`,
		true,
		0,
		true,
	},
	// Array adding and removing
	{
		`Array document add test 1`,
		`{"foo":["bar",5]}`,
		`{"foo":["bar",5,6]}`,
		`[{"op":"add","path":"/foo/-","value":6}]`,
		true,
		0,
		false,
	},
	{
		`Array document add test 2`,
		`{"foo":["bar",5,6]}`,
		`{"foo":["bar",5]}`,
		`[{"op":"remove","path":"/foo/-1"}]`,
		true,
		0,
		false,
	},
	{
		`Array document add test 3`,
		`{"foo":["bar",5,6]}`,
		`{"foo":[5,6]}`,
		`[{"op":"remove","path":"/foo/0"}]`,
		true,
		0,
		false,
	},
	{
		`Array document add test 4`,
		`{"foo":["bar",5]}`,
		`{"foo":[6,"bar",5]}`,
		`[{"op":"add","path":"/foo/0","value":6}]`,
		true,
		0,
		false,
	},
	{
		`Array document add test 5`,
		`{"foo":["bar",5]}`,
		`{"foo":["bar",6,5]}`,
		`[{"op":"add","path":"/foo/1","value":6}]`,
		true,
		0,
		false,
	},
	// Top-level array adding and removing
	{
		`Top-level array document add test 1`,
		`["bar",5]`,
		`["bar",5,6]`,
		`[{"op":"add","path":"/-","value":6}]`,
		true,
		0,
		false,
	},
	{
		`Top-level array document add test 2`,
		`["bar",5,6]`,
		`["bar",5]`,
		`[{"op":"remove","path":"/-1"}]`,
		true,
		0,
		false,
	},
	// Simple copying
	{
		`Copy test 1`,
		`{"foo":5}`,
		`{"foo":5,"bar":5}`,
		`[{"op":"copy","path":"/bar","from":"/foo"}]`,
		true,
		0,
		false,
	},
	{
		`Copy test 2`,
		`{"foo":[5]}`,
		`{"foo":[5],"bar":[5]}`,
		`[{"op":"copy","path":"/bar","from":"/foo"}]`,
		true,
		0,
		false,
	},
	{
		`Copy test 3`,
		`{"foo":{"baz":5}}`,
		`{"foo":{"baz":5},"bar":{"baz":5}}`,
		`[{"op":"copy","path":"/bar","from":"/foo"}]`,
		true,
		0,
		false,
	},
	// Copy and mutate invariance
	{
		`Copy and mutate test 1`,
		`{"foo":5}`,
		`{"foo":5,"bar":6}`,
		`[{"op":"copy","path":"/bar","from":"/foo"},
                  {"op":"replace","path":"/bar","value":6}]`,
		true,
		0,
		false,
	},
	{
		`Copy and mutate test 2`,
		`{"foo":[5]}`,
		`{"foo":[5],"bar":[6]}`,
		`[{"op":"copy","path":"/bar","from":"/foo"},
                  {"op":"replace","path":"/bar/0","value":6}]`,
		true,
		0,
		false,
	},
	{
		`Copy and mutate test 3`,
		`{"foo":{"baz":5}}`,
		`{"foo":{"baz":5},"bar":{"baz":6}}`,
		`[{"op":"copy","path":"/bar","from":"/foo"},
                  {"op":"replace","path":"/bar/baz","value":6}]`,
		true,
		0,
		false,
	},
	// Move tests
	{
		`Move test 1`,
		`{"foo":5}`,
		`{"bar":5}`,
		`[{"op":"move","from":"/foo","path":"/bar"}]`,
		true,
		0,
		false,
	},
	{
		`Move test 2`,
		`{"foo":5}`,
		`{"bar":5}`,
		`[{"op":"move","from":"/foo","path":"/foo/bar"}]`,
		false,
		0,
		false,
	},
	{
		`Move test 3`,
		`{"foo":5}`,
		`{"bar":5}`,
		`[{"op":"move","from":"/foo/5","path":"/bar"}]`,
		false,
		0,
		false,
	},
	// Replace tests
	{
		`Replace test 1`,
		`{"foo":5}`,
		`{"foo":6}`,
		`[{"op":"replace","path":"/foo","value":6}]`,
		true,
		0,
		true,
	},
	{
		`Replace test 2`,
		`{"foo":5}`,
		`{"foo":5}`,
		`[{"op":"replace","path":"/bar","value":6}]`,
		false,
		0,
		false,
	},
	{
		`Replace test 3`,
		`{"foo":5}`,
		`{"bar":5}`,
		`[{"op":"replace","path":"/foo/5","value":6}]`,
		false,
		0,
		false,
	},
	{
		`Replace test 4`,
		`{"foo":5}`,
		`{"bar":5}`,
		`[{"op":"replace","path":"","value":{"bar":5}}]`,
		true,
		0,
		false,
	},
	{
		`Replace test 5`,
		`{"foo":5}`,
		`{"foo":"bar"}`,
		`[{"op":"replace","path":"/foo","value":"bar"}]`,
		true,
		0,
		true,
	},
}

func runTest(t *testing.T, test *opTest, full bool) {
	t.Log(test.desc)
	var src, res, final interface{}
	if err := json.Unmarshal([]byte(test.src), &src); err != nil {
		t.Errorf("`%v` is not a valid JSON source (%v)", test.src, err)
		return
	}
	if err := json.Unmarshal([]byte(test.final), &final); err != nil {
		t.Errorf("`%v` is not a valid JSON final (%v)", test.final, err)
		return
	}
	patch, err := NewPatch([]byte(test.patch))
	if err != nil {
		t.Errorf("%v: Failed to make a Patch: %#v", test.desc, err)
	}
	resBytes, err, idx := patch.Apply([]byte(test.src))
	if test.pass {
		if err != nil {
			t.Errorf("Failed to apply patch `%v`. Failed at operation %v (%v)", test.patch, idx, err)
			return
		}
		if err := json.Unmarshal(resBytes, &res); err != nil {
			t.Errorf("`%v` is not a valid JSON result: %v", string(resBytes), err)
			return
		}
		if !reflect.DeepEqual(res, final) {
			actual, err := json.Marshal(res)
			if err != nil {
				t.Errorf("Failed to make JSON for patched result to display error! (%v)", err)
				return
			}
			t.Errorf("Applying patch `%v` to `%v` did not yield expected result `%v`!", test.patch, test.src, test.final)
			t.Errorf("Got `%v` instead", string(actual))
			return
		}
	} else {
		if err == nil {
			t.Errorf("Expected patch `%v` to fail at operation %v, but it passed.", test.patch, idx)
			return
		} else if idx != test.failidx {
			t.Errorf("Expected patch `%v` to fail at operation %v, but it failed at %v instead!", test.patch, test.failidx, idx)
			return
		}
	}
	if !test.shouldPatch {
		return
	}
	var testPatch Patch
	if full {
		testPatch, err = GenerateFull([]byte(test.src), []byte(test.final), false, true)

	} else {
		testPatch, err = Generate([]byte(test.src), []byte(test.final), false)
	}
	if err != nil {
		t.Errorf("Failed to generate patch to translate `%v` to `%v` (`%v`", test.src, test.final, err)
		return
	}

	if !reflect.DeepEqual(patch, testPatch) {
		buf, _ := json.Marshal(testPatch)
		t.Errorf("Generated patch \n\t`%v` \nis not equal to reference patch \n\t`%v`", string(buf), test.patch)
	}
	newResBytes, err, idx := testPatch.Apply([]byte(test.src))
	if err != nil {
		t.Errorf("Failed to apply generated patch `%v`. Failed at operation %v (%v)", testPatch, idx, err)
		return
	}
	var newRes interface{}
	if err := json.Unmarshal(newResBytes, &newRes); err != nil {
		t.Errorf("`%v` is not a valid JSON result: %v", string(newResBytes), err)
		return
	}
	if !reflect.DeepEqual(newRes, final) {
		actual, err := json.Marshal(res)
		if err != nil {
			t.Errorf("Failed to make JSON for patched result to display error! (%v)", err)
			return
		}
		t.Errorf("Applying generated patch `%v` to `%v` did not yield expected result `%v`!", testPatch, test.src, test.final)
		t.Errorf("Got `%v` instead", string(actual))
		return
	}
}

func TestPatches(t *testing.T) {
	for _, test := range opTests {
		runTest(t, &test, false)
	}
}

var fullOpTests = []opTest{
	// Object adding and removing
	{
		`Basic document add test 1`,
		`{"foo":["bar",5]}`,
		`{"foo":["bar",5],"bar":5}`,
		`[{"op":"test","path":"","from":"","value":{"foo":["bar",5]}},{"op":"add","path":"/bar","from":"","value":5}]`,
		true,
		0,
		true,
	},
	{
		`Basic document add test 2`,
		`{"foo":["bar",5],"bar":5}`,
		`{"foo":["bar",5]}`,
		`[{"op":"test","path":"","from":"","value":{"bar":5,"foo":["bar",5]}},{"op":"remove","path":"/bar","from":"","value":null}]`,
		true,
		0,
		true,
	},

	// Nested object adding and removing
	{
		`Nested document add test 1`,
		`{"foo":{"bar":5}}`,
		`{"foo":{"bar":5,"baz":6}}`,
		`[{"op":"test","path":"","from":"","value":{"foo":{"bar":5}}},{"op":"add","path":"/foo/baz","from":"","value":6}]`,
		true,
		0,
		true,
	},
	{
		`Nested document add test 2`,
		`{"foo":{"bar":5,"baz":6}}`,
		`{"foo":{"bar":5}}`,
		`[{"op":"test","path":"","from":"","value":{"foo":{"bar":5,"baz":6}}},{"op":"remove","path":"/foo/baz","from":"","value":null}]`,
		true,
		0,
		true,
	},
	// Replace tests
	{
		`Replace test 1`,
		`{"foo":5}`,
		`{"foo":6}`,
		`[{"op":"test","path":"","from":"","value":{"foo":5}},{"op":"replace","path":"/foo","from":"","value":6}]`,
		true,
		0,
		true,
	},
	{
		`Replace test 5`,
		`{"foo":5}`,
		`{"foo":"bar"}`,
		`[{"op":"test","path":"","from":"","value":{"foo":5}},{"op":"replace","path":"/foo","from":"","value":"bar"}]`,
		true,
		0,
		true,
	},
}

func TestFullPatches(t *testing.T) {
	for _, test := range fullOpTests {
		runTest(t, &test, true)
	}
}
