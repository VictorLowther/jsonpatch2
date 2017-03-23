package jsonpatch2

import (
	"encoding/json"
	"reflect"

	"github.com/VictorLowther/jsonpatch2/utils"
)

// This generator does not create copy or move patch ops, and I don't
// care enough to optimize it to do so.  Ditto for slice handling.
// There is a lot of optimization that could be done here, but it can get complex real quick.
func basicGen(base, target interface{}, paranoid bool, ptr pointer) Patch {
	res := make(Patch, 0)
	pstr := ptr.String()
	if reflect.TypeOf(base) != reflect.TypeOf(target) {
		if paranoid {
			res = append(res, Operation{"test", pstr, "", utils.Clone(base), ptr, nil})
		}
		res = append(res, Operation{"replace", pstr, "", utils.Clone(target), ptr, nil})
		return res
	}
	switch baseVal := base.(type) {
	case map[string]interface{}:
		targetVal := target.(map[string]interface{})
		handled := make(map[string]struct{})
		// Handle removed and changed first.
		for k, oldVal := range baseVal {
			newPtr := ptr.Append(k)
			newPstr := newPtr.String()
			newVal, ok := targetVal[k]
			if !ok {
				// Generate a remove op
				if paranoid {
					res = append(res, Operation{"test", newPstr, "", utils.Clone(oldVal), newPtr, nil})
				}
				res = append(res, Operation{"remove", newPstr, "", nil, newPtr, nil})
			} else {
				subPatch := basicGen(oldVal, newVal, paranoid, newPtr)
				res = append(res, subPatch...)
			}
			handled[k] = struct{}{}
		}
		// Now, handle additions
		for k, newVal := range targetVal {
			if _, ok := handled[k]; ok {
				continue
			}
			newPtr := ptr.Append(k)
			newPstr := newPtr.String()
			res = append(res, Operation{"add", newPstr, "", utils.Clone(newVal), newPtr, nil})
		}
	// case []interface{}:
	// Eventually, add code to handle slices more
	// efficiently.  For now, through, be dumb.
	default:
		if !reflect.DeepEqual(base, target) {
			if paranoid {
				res = append(res, Operation{"test", pstr, "", utils.Clone(base), ptr, nil})
			}
			res = append(res, Operation{"replace", pstr, "", utils.Clone(target), ptr, nil})
		}
	}
	return res
}

// Generate generates a JSON Patch that will modify base into target.
// If paranoid is true, then the generated patch will have test checks.
//
// base and target must be byte arrays containing valid JSON
func Generate(base, target []byte, paranoid bool) (Patch, error) {
	var rawBase, rawTarget interface{}
	if err := json.Unmarshal(base, &rawBase); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(target, &rawTarget); err != nil {
		return nil, err
	}
	return basicGen(rawBase, rawTarget, paranoid, make(pointer, 0)), nil
}
