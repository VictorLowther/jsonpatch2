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
	Path Pointer `json:"path"`
	// From is a JSON pointer indicating where a value should be
	// copied/moved from.  From is only used by copy and move operations.
	From Pointer `json:"from"`
	// Value is the Value to be used for add, replace, and test operations.
	Value interface{} `json:"value"`
}

func (o *Operation) MarshalJSON() ([]byte, error) {
	res := map[string]interface{}{}
	res["op"] = o.Op
	res["path"] = o.Path
	switch o.Op {
	case "copy", "move":
		res["from"] = o.From
	case "add", "replace", "test":
		res["value"] = o.Value
	}
	return json.Marshal(res)
}

const ContentType = "application/json-patch+json"

// Apply performs a single patch operation
func (o *Operation) Apply(to interface{}) (interface{}, error) {
	switch o.Op {
	case "test":
		return to, o.Path.Test(to, o.Value)
	case "replace":
		return o.Path.Replace(to, o.Value)
	case "add":
		return o.Path.Put(to, o.Value)
	case "remove":
		return o.Path.Remove(to)
	case "move":
		return o.From.Move(to, o.Path)
	case "copy":
		return o.From.Copy(to, o.Path)
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
		if op.Path == nil {
			return res, fmt.Errorf("Did not get valid path")
		}
		switch op.Op {
		case "test":
			fallthrough
		case "replace":
			fallthrough
		case "add":
			if op.Value == nil {
				return res, fmt.Errorf("%v must have a valid value", op.Op)

			}
		case "move":
			fallthrough
		case "copy":
			if op.From == nil {
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

// Apply applies rawPatch (which must be a []byte containing a valid
// JSON Patch) to base, yielding result.  If err is returned, the
// returned int is the index of the operation that failed.  If the
// error is that rawPatch is not a valid JSON Patch, loc will be 0,
//
// base must be the result of unmarshaling JSON to interface{}, and
// will not be modified.
func (p Patch) Apply(base interface{}) (result interface{}, err error, loc int) {
	result = utils.Clone(base)
	for i, op := range p {
		result, err = op.Apply(result)
		if err != nil {
			return result, err, i
		}
	}
	return result, nil, 0
}

// ApplyJSON does the same thing as Apply, except the inputs should be
// JSON-containing byte arrays instead of unmarshalled JSON
func (p Patch) ApplyJSON(base []byte) (result []byte, err error, loc int) {
	var rawBase interface{}
	err = json.Unmarshal(base, &rawBase)
	if err != nil {
		return nil, err, 0
	}
	rawRes, err, loc := p.Apply(rawBase)
	if err != nil {
		return nil, err, loc
	}
	result, err = json.Marshal(rawRes)
	return result, err, loc
}
