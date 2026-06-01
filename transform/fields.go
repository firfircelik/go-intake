package transform

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/firfircelik/go-intake"
)

// ParseFloat returns a Transformer that converts the string value in
// field to a float64. Empty or whitespace-only values are cleared.
// Other unparseable values produce an error that includes the
// field name. Missing or non-string values are left untouched.
//
// Apply never mutates the input record: it always returns a fresh
// Record. Mutating the returned Record does not affect the input.
func ParseFloat(field string) intake.Transformer {
	return &parseFloat{field: field}
}

type parseFloat struct {
	field string
}

func (p *parseFloat) Apply(_ context.Context, r intake.Record) (intake.Record, error) {
	if r == nil {
		return nil, nil
	}
	v, ok := r[p.field]
	if !ok {
		return clone(r), nil
	}
	s, ok := v.(string)
	if !ok {
		return clone(r), nil
	}
	s = strings.TrimSpace(s)
	if s == "" {
		out := clone(r)
		delete(out, p.field)
		return out, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return clone(r), fmt.Errorf("parse float: field %q: %w", p.field, err)
	}
	out := clone(r)
	out[p.field] = f
	return out, nil
}

// ParseInt returns a Transformer that converts the string value in
// field to an int64. Empty or whitespace-only values are cleared.
// Other unparseable values produce an error that includes the
// field name. Missing or non-string values are left untouched.
//
// Apply never mutates the input record: it always returns a fresh
// Record. Mutating the returned Record does not affect the input.
func ParseInt(field string) intake.Transformer {
	return &parseInt{field: field}
}

type parseInt struct {
	field string
}

func (p *parseInt) Apply(_ context.Context, r intake.Record) (intake.Record, error) {
	if r == nil {
		return nil, nil
	}
	v, ok := r[p.field]
	if !ok {
		return clone(r), nil
	}
	s, ok := v.(string)
	if !ok {
		return clone(r), nil
	}
	s = strings.TrimSpace(s)
	if s == "" {
		out := clone(r)
		delete(out, p.field)
		return out, nil
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return clone(r), fmt.Errorf("parse int: field %q: %w", p.field, err)
	}
	out := clone(r)
	out[p.field] = i
	return out, nil
}

// ParseBool returns a Transformer that converts the string value in
// field to a bool. Recognised truthy values are "1", "t", "true",
// "y", "yes" (case-insensitive). Recognised falsy values are "0",
// "f", "false", "n", "no" (case-insensitive). Empty or
// whitespace-only values are cleared. Other values produce an error
// that includes the field name. Missing or non-string values are
// left untouched.
//
// Apply never mutates the input record: it always returns a fresh
// Record. Mutating the returned Record does not affect the input.
func ParseBool(field string) intake.Transformer {
	return &parseBool{field: field}
}

type parseBool struct {
	field string
}

func (p *parseBool) Apply(_ context.Context, r intake.Record) (intake.Record, error) {
	if r == nil {
		return nil, nil
	}
	v, ok := r[p.field]
	if !ok {
		return clone(r), nil
	}
	s, ok := v.(string)
	if !ok {
		return clone(r), nil
	}
	s = strings.TrimSpace(s)
	if s == "" {
		out := clone(r)
		delete(out, p.field)
		return out, nil
	}
	b, err := parseBoolString(s)
	if err != nil {
		return clone(r), fmt.Errorf("parse bool: field %q: %w", p.field, err)
	}
	out := clone(r)
	out[p.field] = b
	return out, nil
}

func parseBoolString(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "1", "t", "true", "y", "yes":
		return true, nil
	case "0", "f", "false", "n", "no":
		return false, nil
	}
	return false, fmt.Errorf("unrecognised boolean value %q", s)
}

// ParseDate returns a Transformer that parses the string value in
// field using the supplied time.Parse layouts, tried in order. The
// first layout that successfully parses the (trimmed) value wins,
// and the result is stored as a time.Time. Empty or whitespace-only
// values are cleared. Other values produce an error that includes
// the field name. Missing or non-string values are left untouched.
// If layouts is empty, every Apply returns an error that includes
// the field name.
//
// Apply never mutates the input record: it always returns a fresh
// Record. Mutating the returned Record does not affect the input.
func ParseDate(field string, layouts ...string) intake.Transformer {
	return &parseDate{field: field, layouts: layouts}
}

type parseDate struct {
	field   string
	layouts []string
}

func (p *parseDate) Apply(_ context.Context, r intake.Record) (intake.Record, error) {
	if r == nil {
		return nil, nil
	}
	if len(p.layouts) == 0 {
		return clone(r), fmt.Errorf("parse date: field %q: no layouts provided", p.field)
	}
	v, ok := r[p.field]
	if !ok {
		return clone(r), nil
	}
	s, ok := v.(string)
	if !ok {
		return clone(r), nil
	}
	s = strings.TrimSpace(s)
	if s == "" {
		out := clone(r)
		delete(out, p.field)
		return out, nil
	}
	for _, layout := range p.layouts {
		if t, err := time.Parse(layout, s); err == nil {
			out := clone(r)
			out[p.field] = t
			return out, nil
		}
	}
	return clone(r), fmt.Errorf("parse date: field %q: no layout matched value %q", p.field, s)
}

// Rename returns a Transformer that renames the field from to to.
// If to already exists, the existing value is overwritten. If from
// does not exist, the record is returned with a fresh map but the
// target field is not added. Renaming a field to itself is a no-op.
//
// Apply never mutates the input record: it always returns a fresh
// Record. Mutating the returned Record does not affect the input.
func Rename(from, to string) intake.Transformer {
	return renameField{from: from, to: to}
}

type renameField struct {
	from string
	to   string
}

func (r renameField) Apply(_ context.Context, rec intake.Record) (intake.Record, error) {
	if rec == nil {
		return nil, nil
	}
	v, ok := rec[r.from]
	if !ok {
		return clone(rec), nil
	}
	if r.from == r.to {
		return clone(rec), nil
	}
	out := clone(rec)
	delete(out, r.from)
	out[r.to] = v
	return out, nil
}

// Drop returns a Transformer that removes the named fields from
// each record. Missing fields are silently ignored. Calling Drop
// with no fields returns a shallow copy of the record.
//
// Apply never mutates the input record: it always returns a fresh
// Record. Mutating the returned Record does not affect the input.
func Drop(fields ...string) intake.Transformer {
	return dropFields{fields: fields}
}

type dropFields struct {
	fields []string
}

func (d dropFields) Apply(_ context.Context, r intake.Record) (intake.Record, error) {
	if r == nil {
		return nil, nil
	}
	out := clone(r)
	for _, f := range d.fields {
		delete(out, f)
	}
	return out, nil
}

// Keep returns a Transformer that removes every field not in keep.
// Calling Keep with no fields yields an empty record.
//
// Apply never mutates the input record: it always returns a fresh
// Record. Mutating the returned Record does not affect the input.
func Keep(keep ...string) intake.Transformer {
	return keepFields{keep: keep}
}

type keepFields struct {
	keep []string
}

func (k keepFields) Apply(_ context.Context, r intake.Record) (intake.Record, error) {
	if r == nil {
		return nil, nil
	}
	out := make(intake.Record, len(k.keep))
	for _, f := range k.keep {
		if v, ok := r[f]; ok {
			out[f] = v
		}
	}
	return out, nil
}

// AddField returns a Transformer that sets field to value on every
// record. Existing values are overwritten. The value is stored as
// the supplied any, including nil if value is nil.
//
// Apply never mutates the input record: it always returns a fresh
// Record. Mutating the returned Record does not affect the input.
func AddField(field string, value any) intake.Transformer {
	return addField{field: field, value: value}
}

type addField struct {
	field string
	value any
}

func (s addField) Apply(_ context.Context, r intake.Record) (intake.Record, error) {
	if r == nil {
		return nil, nil
	}
	out := clone(r)
	out[s.field] = s.value
	return out, nil
}

// Copy returns a Transformer that copies the value of from into to.
// Unlike Rename, the source field is preserved. If from does not
// exist, the record is returned with a fresh map but the target
// field is not added. Copying a field to itself leaves the value
// unchanged.
//
// Apply never mutates the input record: it always returns a fresh
// Record. Mutating the returned Record does not affect the input.
func Copy(from, to string) intake.Transformer {
	return copyField{from: from, to: to}
}

type copyField struct {
	from string
	to   string
}

func (c copyField) Apply(_ context.Context, r intake.Record) (intake.Record, error) {
	if r == nil {
		return nil, nil
	}
	v, ok := r[c.from]
	if !ok {
		return clone(r), nil
	}
	out := clone(r)
	out[c.to] = v
	return out, nil
}

// MapField returns a Transformer that applies fn to the value of
// field and stores the result back. fn receives the current value
// (including nil) and may return any value. If fn returns an error,
// the error is wrapped to include the field name and returned. If
// field does not exist, the record is returned with a fresh map and
// the field is not added.
//
// Apply never mutates the input record: it always returns a fresh
// Record. Mutating the returned Record does not affect the input.
func MapField(name string, fn func(any) (any, error)) intake.Transformer {
	return mapField{name: name, fn: fn}
}

type mapField struct {
	name string
	fn   func(any) (any, error)
}

func (m mapField) Apply(_ context.Context, r intake.Record) (intake.Record, error) {
	if r == nil {
		return nil, nil
	}
	v, ok := r[m.name]
	if !ok {
		return clone(r), nil
	}
	nv, err := m.fn(v)
	if err != nil {
		return clone(r), fmt.Errorf("map field: field %q: %w", m.name, err)
	}
	out := clone(r)
	out[m.name] = nv
	return out, nil
}

// clone returns a shallow copy of r. If r is nil, clone returns nil.
func clone(r intake.Record) intake.Record {
	if r == nil {
		return nil
	}
	out := make(intake.Record, len(r))
	for k, v := range r {
		out[k] = v
	}
	return out
}
