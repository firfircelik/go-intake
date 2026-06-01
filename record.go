package intake

// Record is a single row of record-oriented data.
//
// A Record is a string-keyed map of arbitrary values. Values are
// typically JSON-compatible scalars (string, float64, bool, nil) or
// nested Records and slices thereof.
//
// Records are treated as immutable by transformers: a Transformer's
// Apply method returns a new Record (it may return the same one for
// in-place transforms) and the pipeline never mutates a Record it
// has already passed downstream.
type Record map[string]any

// Get returns the value for key and reports whether the key was
// present. The bool distinguishes "missing" from "present with
// zero value". It is safe to call on a nil Record.
func (r Record) Get(key string) (any, bool) {
	if r == nil {
		return nil, false
	}
	v, ok := r[key]
	return v, ok
}

// Has reports whether the record contains a value for key. It is
// safe to call on a nil Record.
func (r Record) Has(key string) bool {
	if r == nil {
		return false
	}
	_, ok := r[key]
	return ok
}
