// Package enrich provides lookup and enrichment transformers.
package enrich

import (
	"context"
	"fmt"
	"sync"

	"github.com/firfircelik/go-intake"
)

// CacheLookup enriches records with cached lookup values.
type CacheLookup struct {
	lookupField string
	cache       map[string]any
	lockField   string
	m           sync.RWMutex
}

// NewCacheLookup creates a lookup transformer.
// lookupField is the field to look up, cache contains the lookup values.
func NewCacheLookup(lookupField string, cache map[string]any) *CacheLookup {
	return &CacheLookup{
		lookupField: lookupField,
		cache:       cache,
	}
}

// Apply enriches the record with lookup values.
func (l *CacheLookup) Apply(ctx context.Context, r intake.Record) (intake.Record, error) {
	val, exists := r.Get(l.lookupField)
	if !exists {
		return r, nil
	}

	l.m.RLock()
	lookupVal, found := l.cache[fmt.Sprintf("%v", val)]
	l.m.RUnlock()

	if found {
		// Create new record to maintain immutability
		newRec := make(intake.Record)
		for k, v := range r {
			newRec[k] = v
		}
		for k, v := range lookupVal.(map[string]any) {
			newRec[k] = v
		}
		return newRec, nil
	}
	return r, nil
}

// StaticMapEnrich enriches using a static map.
type StaticMapEnrich struct {
	field    string
	mapField string
	mapping  map[string]string
}

// NewStaticMapEnrich creates a static mapper.
func NewStaticMapEnrich(field, mapField string, mapping map[string]string) *StaticMapEnrich {
	return &StaticMapEnrich{
		field:    field,
		mapField: mapField,
		mapping:  mapping,
	}
}

// Apply transforms the record using the mapping.
func (e *StaticMapEnrich) Apply(ctx context.Context, r intake.Record) (intake.Record, error) {
	val, exists := r.Get(e.field)
	if !exists {
		return r, nil
	}

	mapped, found := e.mapping[fmt.Sprintf("%v", val)]
	if !found {
		return r, nil
	}

	// Create new record
	newRec := make(intake.Record)
	for k, v := range r {
		newRec[k] = v
	}
	newRec[e.mapField] = mapped
	return newRec, nil
}
