package jsonpatch2

// jsonpatch is a library for creating and applying JSON Patches as defined in RFC 6902.
//
// A JSON patch is a list of operations in the form of:
//    [
//      {"op":"test","path":"/foo","value":"bar"},
//      {"op":"replace","path":"/foo","value":"baz"}
//      ...
//    ]
//
// See http://tools.ietf.org/html/rfc6902 for more information.

import (
	"encoding/json"
	"fmt"

	"github.com/VictorLowther/jsonpatch2/utils"
)

// operation represents a valid JSON Patch operation as defined by RFC 6902
type Operation struct {
	// Op can be one of:
	//    * "add"
	//    * "remove"
	//    * "replace"
	//    * "move"
	//    * "copy"
	//    * "test"
	// All Operations must have an Op.
	Op string `json:"op"`
	// Path is a JSON Pointer as defined in RFC 6901
	// All Operations must have a Path
	Path string `json:"path"`
	// From is a JSON pointer indicating where a value should be
	// copied/moved from.  From is only used by copy and move operations.
	From string `json:"from"`
	// Value is the Value to be used for add, replace, and test operations.
	Value      interface{} `json:"value"`
	path, from pointer
}

func (o *Operation) UnmarshalJSON(buf []byte) error {
	type op struct {
		Op    string      `json:"op"`
		Path  string      `json:"path"`
		From  string      `json:"from"`
		Value interface{} `json:"value"`
	}
	ref := op{}
	if err := json.Unmarshal(buf, &ref); err != nil {
		return err
	}
	o.Op, o.Path, o.From, o.Value = ref.Op, ref.Path, ref.From, ref.Value
	path, err := newPointer(o.Path)
	if err != nil {
		return err
	}
	o.path = path
	switch o.Op {
	case "copy", "move":
		from, err := newPointer(o.From)
		if err != nil {
			return err
		}
		o.from = from
	}
	return nil
}

const ContentType = "application/json-patch+json"

// Apply performs a single patch operation
func (o *Operation) apply(to interface{}) (interface{}, error) {
	switch o.Op {
	case "test":
		return to, o.path.Test(to, o.Value)
	case "replace":
		return o.path.Replace(to, o.Value)
	case "add":
		return o.path.Put(to, o.Value)
	case "remove":
		return o.path.Remove(to)
	case "move":
		return o.from.Move(to, o.path)
	case "copy":
		return o.from.Copy(to, o.path)
	default:
		return to, fmt.Errorf("Invalid op %v", o.Op)
	}
}

// Patch is an array of individual JSON Patch operations.
type Patch []Operation

// NewPatch takes a byte array and tries to unmarshal it.
func NewPatch(buf []byte) (res Patch, err error) {
	res = make(Patch, 0)
	if err = json.Unmarshal(buf, &res); err != nil {
		return nil, err
	}

	for _, op := range res {
		if op.path == nil {
			return res, fmt.Errorf("Did not get valid path")
		}
		switch op.Op {
		case "test", "replace", "add":
			if op.Value == nil {
				return res, fmt.Errorf("%v must have a valid value", op.Op)

			}
		case "move", "copy":
			if op.from == nil {
				return res, fmt.Errorf("%v must have a from", op.Op)
			}
		case "remove":
			continue
		default:
			return res, fmt.Errorf("%v is not a valid JSON Patch operator", op.Op)
		}
	}
	return res, nil
}

func (p Patch) apply(base interface{}) (result interface{}, err error, loc int) {
	result = utils.Clone(base)
	for i, op := range p {
		result, err = op.apply(result)
		if err != nil {
			return result, err, i
		}
	}
	return result, nil, 0
}

// Apply applies p to base (which must be a byte array containing
// valid JSON), yielding result (which will also be a byte array
// containing valid JSON).  If err is returned, the returned int is
// the index of the operation that failed.

// ApplyJSON does the same thing as Apply, except the inputs should be
// JSON-containing byte arrays instead of unmarshalled JSON
func (p Patch) Apply(base []byte) (result []byte, err error, loc int) {
	var rawBase interface{}
	err = json.Unmarshal(base, &rawBase)
	if err != nil {
		return nil, err, 0
	}
	rawRes, err, loc := p.apply(rawBase)
	if err != nil {
		return nil, err, loc
	}
	result, err = json.Marshal(rawRes)
	return result, err, loc
}
