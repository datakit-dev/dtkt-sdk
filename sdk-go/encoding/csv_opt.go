package encoding

import "encoding/csv"

type (
	CSVWriterOption func(*csv.Writer)
	CSVReaderOption func(*csv.Reader)
)

// Field delimiter (set to ',' by default)
func WithCSVReaderComma(comma rune) CSVReaderOption {
	return func(r *csv.Reader) {
		r.Comma = comma
	}
}

// If LazyQuotes is true, a quote may appear in an unquoted field and a
// non-doubled quote may appear in a quoted field.
func WithCSVReaderLazyQuotes(lazyQuotes bool) CSVReaderOption {
	return func(r *csv.Reader) {
		r.LazyQuotes = lazyQuotes
	}
}

// If TrimLeadingSpace is true, leading white space in a field is ignored.
// This is done even if the field delimiter, Comma, is white space.
func WithCSVReaderTrimLeadingSpace(trimSpace bool) CSVReaderOption {
	return func(r *csv.Reader) {
		r.TrimLeadingSpace = trimSpace
	}
}

// ReuseRecord controls whether calls to Read may return a slice sharing the
// backing array of the previous call's returned slice for performance. By
// default, each call to Read returns newly allocated memory owned by the caller.
func WithCSVReaderReuseRecord(reuse bool) CSVReaderOption {
	return func(r *csv.Reader) {
		r.ReuseRecord = reuse
	}
}

// FieldsPerRecord is the number of expected fields per record.
// If FieldsPerRecord is positive, Read requires each record to
// have the given number of fields. If FieldsPerRecord is 0, Read sets it to
// the number of fields in the first record, so that future records must
// have the same field count. If FieldsPerRecord is negative, no check is
// made and records may have a variable number of fields.
func WithCSVReaderFieldCount(count int) CSVReaderOption {
	return func(r *csv.Reader) {
		r.FieldsPerRecord = count
	}
}

// Field delimiter (set to ',' by default)
func WithCSVWriterComma(comma rune) CSVWriterOption {
	return func(w *csv.Writer) {
		w.Comma = comma
	}
}

// Set true to use "\r\n" as the line terminator
func WithCSVWriterUseCRLF(useCRLF bool) CSVWriterOption {
	return func(w *csv.Writer) {
		w.UseCRLF = useCRLF
	}
}
