package jsonpatch2

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/VictorLowther/jsonpatch2/utils"
)

// pointerSegment is an individual fragment of a pointer.
type pointerSegment string

var decode = strings.NewReplacer("~1", "/", "~0", "~")
var encode = strings.NewReplacer("~", "~0", "/", "~1")

// String translates a pointerSegment into a regular string, encoding it as we go.
func (s pointerSegment) String() string {
	return encode.Replace(string(s))
}

// NewSegment creates a new segment from an encoded string.
func newSegment(s string) (pointerSegment, error) {
	c := strings.Split(s, `~`)
	if len(c) != 0 {
		for i := 1; i < len(c); i++ {
			if strings.HasPrefix(c[i], `0`) || strings.HasPrefix(c[i], `1`) {
				continue
			}
			return pointerSegment(""), fmt.Errorf("`%s` has an illegal unescaped ~", s)
		}
	}
	return pointerSegment(decode.Replace(s)), nil
}

// Pointer is a JSON Pointer
type Pointer []pointerSegment

// Contains tests to see if B is a subset of A
func (a Pointer) Contains(b Pointer) bool {
	return len(b) >= len(a) && reflect.DeepEqual(a, b[:len(a)])
}

// newpointer takes a string that conforms to RFC6901 and turns it into a JSON pointer.
func NewPointer(s string) (Pointer, error) {
	frags := strings.Split(s, `/`)[1:]
	res := make(Pointer, len(frags))
	// An empty pointer refers to the whole document, and so is valid.
	if s == "" {
		return res, nil
	}
	if !strings.HasPrefix(s, "/") {
		return nil, fmt.Errorf("Initial character of a non-empty pointer must be `/`")
	}
	for i, frag := range frags {
		q, err := newSegment(frag)
		if err != nil {
			return nil, err
		}
		res[i] = q
	}
	return res, nil
}

// Allow a pointer to be marshalled to valid JSON.
func (p Pointer) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

// Allow unmarshalling from JSON
func (p *Pointer) UnmarshalJSON(buf []byte) error {
	var b string
	if err := json.Unmarshal(buf, &b); err != nil {
		return err
	}
	ptr, err := NewPointer(b)
	*p = ptr[:]
	return err
}

// String takes a pointer and returns its string value.
func (p Pointer) String() string {
	frags := make([]string, len(p)+1)
	for i, frag := range p {
		frags[i+1] = frag.String()
	}
	return strings.Join(frags, `/`)
}

// Shift extracts the first element in the pointer, returning it and the rest of the pointer.
func (p Pointer) Shift() (string, Pointer) {
	if len(p) == 0 {
		return "", p
	}
	return string(p[0]), Pointer(p[1:])
}

// Chop extracts the last element in the pointer, returning it and the rest of the pointer.
func (p Pointer) Chop() (string, Pointer) {
	if len(p) == 0 {
		return "", p
	}
	last := len(p) - 1
	return string(p[last]), Pointer(p[:last])
}

func (p Pointer) Append(frag string) Pointer {
	return append(p, pointerSegment(decode.Replace(frag)))
}

func normalizeOffset(selector string, bound int) (int, error) {
	res, err := strconv.Atoi(selector)
	if err != nil {
		return -1, err
	}
	if res < 0 {
		res = bound + res
	}
	if res >= bound || res < 0 {
		return -1, fmt.Errorf("Index out of bounds")
	}
	return res, nil
}

// Get takes an unmarshalled JSON blob, and returns the value pointed at by the pointer.
// The unmarshalled blob is left unchanged.
func (p Pointer) Get(from interface{}) (interface{}, error) {
	if len(p) == 0 {
		return from, nil
	}
	selector, nextPointer := p.Shift()
	switch t := from.(type) {
	case map[string]interface{}:
		found, ok := t[selector]
		if !ok {
			return nil, fmt.Errorf("Selector %v not a member of %#v", selector, t)
		}
		return nextPointer.Get(found)
	case []interface{}:
		index, err := normalizeOffset(selector, len(t))
		if err != nil {
			return nil, err
		}
		return nextPointer.Get(t[index])
	default:
		return nil, fmt.Errorf("Cannot index pointer %v for non-indexable JSON value", p.String())
	}
}

func (p Pointer) toContainer(to interface{}) (string, interface{}, error) {
	if len(p) == 0 {
		return "", nil, fmt.Errorf("Cannot happen")
	}
	selector, getPointer := p.Chop()
	operatrix, err := getPointer.Get(to)
	return selector, operatrix, err
}

// Replace replaces the pointed at value (which must exist) with val.
func (p Pointer) Replace(to interface{}, val interface{}) (interface{}, error) {
	if len(p) == 0 {
		return val, nil
	}
	selector, operatrix, err := p.toContainer(to)
	if err != nil {
		return to, err
	}
	switch t := operatrix.(type) {
	case map[string]interface{}:
		if _, ok := t[selector]; ok {
			t[selector] = val
		} else {
			return to, fmt.Errorf("%v does not refer to an existing location", p.String())
		}
	case []interface{}:
		index, err := normalizeOffset(selector, len(t))
		if err != nil {
			return to, err
		}
		t[index] = val
	default:
		return to, fmt.Errorf("Cannot put to non-indexable JSON value")
	}
	return to, nil
}

func (p Pointer) handleChangedSlice(to interface{}, s []interface{}) (interface{}, error) {
	if len(p) > 1 {
		_, holdPtr := p.Chop()
		return holdPtr.Replace(to, s)
	} else {
		return s, nil
	}
}

// Put puts val into to at the position indicated by the pointer,
// returning a possibly new value for to.  The position does not have
// to already exist or refer to a preexisting Value.
//
// Put may have to return a new to if to happens to be a slice, since
// the semantics of Put necessarily involve growing the Slice.
func (p Pointer) Put(to interface{}, val interface{}) (interface{}, error) {
	selector, operatrix, err := p.toContainer(to)
	if err != nil {
		return to, err
	}
	switch t := operatrix.(type) {
	case map[string]interface{}:
		t[selector] = val
	case []interface{}:
		if selector == "-" {
			t = append(t, val)
		} else {
			index, err := normalizeOffset(selector, len(t))
			if err != nil {
				return to, err
			}
			res := make([]interface{}, len(t)+1)
			k := res[index+1:]
			copy(res, t[:index])
			copy(k, t[index:])
			res[index] = val
			t = res
		}
		return p.handleChangedSlice(to, t)
	default:
		return to, fmt.Errorf("Cannot put to non-indexable JSON value")
	}
	return to, nil
}

// Remove removes the value pointed to by the pointer from from,
// returning a possibly new value for from.
//
// Remove may have to return a new from if it is a slice, because the
// semantics for Reomve on a Slice involve shrinking it, which
// involves reallocation the way we do it.
func (p *Pointer) Remove(from interface{}) (interface{}, error) {
	selector, operatrix, err := p.toContainer(from)
	if err != nil {
		return from, err
	}
	switch t := operatrix.(type) {
	case map[string]interface{}:
		if _, ok := t[selector]; !ok {
			return from, fmt.Errorf("`%v` does not point to an existing location", p.String())
		}
		delete(t, selector)
	case []interface{}:
		index, err := normalizeOffset(selector, len(t))
		if err != nil {
			return from, err
		}
		// Shift everything after our target over by one.
		k := t[index:]
		k2 := t[index+1:]
		copy(k, k2)
		t = t[:len(t)-1]
		return p.handleChangedSlice(from, t)
	default:
		return from, fmt.Errorf("Cannot remove non-indexable JSON value")
	}
	return from, nil
}

// Copy deep-copies the value pointed to by p in from to the location pointed to by at.
func (p Pointer) Copy(from interface{}, at Pointer) (interface{}, error) {
	val, err := p.Get(from)
	if err != nil {
		return from, err
	}
	return at.Put(from, utils.Clone(val))
}

// Move moves the value pointed to by p in from to the location pointed to by at.
func (p Pointer) Move(from interface{}, at Pointer) (interface{}, error) {
	val, err := p.Get(from)
	if err != nil {
		return from, err
	}
	val, err = at.Put(from, val)
	if err != nil {
		return val, err
	}
	return p.Remove(val)
}

func (p *Pointer) Test(from interface{}, sample interface{}) error {
	val, err := p.Get(from)
	if err == nil && !reflect.DeepEqual(val, sample) {
		err = fmt.Errorf("Test op failed.")
	}
	return err
}
