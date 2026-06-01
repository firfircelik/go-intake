// Package transform provides Transformers for intake pipelines.
//
// A Transformer takes a Record and returns a (possibly modified)
// Record. Every transformer in this package honours the no-mutation
// contract documented on intake.Transformer: Apply never mutates
// the input record, and always returns a fresh Record, even on
// no-op, on error, and when the target field is missing.
//
// The package exports three families of transformer:
//
//   - Header style transformers: NormalizeHeaders, which rewrites
//     every record key using a HeaderStyle (SnakeCase, CamelCase,
//     KebabCase, LowerCase, UpperCase, or a custom function).
//   - String transformers: TrimStrings, LowerStrings, UpperStrings.
//     Each accepts an optional field list; when the list is empty,
//     every string value in the record is processed.
//   - Field transformers: ParseFloat, ParseInt, ParseBool, ParseDate,
//     Rename, Drop, Keep, AddField, Copy, MapField. The parsers
//     convert string values to typed values; Rename/Copy/Drop/Keep
//     reshape the record's key set; AddField attaches a constant
//     value; MapField applies a user function to one field's value.
//
// Parse errors from the parsers are reported as errors from Apply.
// The pipeline routes the original (untransformed) record to the
// Quarantine and continues with the next record.
package transform
