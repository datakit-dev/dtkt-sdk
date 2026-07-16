package encoding

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"strconv"
	"sync"

	"google.golang.org/protobuf/types/known/structpb"
)

type (
	CSVEncoder struct {
		writerOpts []CSVWriterOption
		getWriter  func() (*bytes.Buffer, *csv.Writer)
	}
	CSVDecoder struct {
		opts      []CSVReaderOption
		getReader func() (*bytes.Buffer, *csv.Reader)
	}
)

func NewCSVEncoder(opts ...CSVWriterOption) (enc CSVEncoder) {
	enc.writerOpts = opts
	enc.getWriter = sync.OnceValues(func() (buf *bytes.Buffer, writer *csv.Writer) {
		buf = new(bytes.Buffer)
		writer = NewCSVWriter(buf, opts...)
		return
	})
	return
}

func NewCSVDecoder(opts ...CSVReaderOption) (dec CSVDecoder) {
	dec.opts = opts
	dec.getReader = sync.OnceValues(func() (buf *bytes.Buffer, reader *csv.Reader) {
		buf = new(bytes.Buffer)
		reader = NewCSVReader(buf, opts...)
		return
	})
	return
}

func NewCSVWriter(w io.Writer, opts ...CSVWriterOption) *csv.Writer {
	writer := csv.NewWriter(w)
	for _, opt := range opts {
		if opt != nil {
			opt(writer)
		}
	}
	return writer
}

func NewCSVReader(r io.Reader, opts ...CSVReaderOption) *csv.Reader {
	reader := csv.NewReader(r)
	for _, opt := range opts {
		if opt != nil {
			opt(reader)
		}
	}
	return reader
}

func (e CSVEncoder) StreamEncode(w io.Writer) func(any) error {
	writer := NewCSVWriter(w, e.writerOpts...)

	return func(v any) error {
		return writeCSVRecord(writer, v)
	}
}

func (e CSVEncoder) Encode(v any) ([]byte, error) {
	buf, writer := e.getWriter()

	err := writeCSVRecord(writer, v)
	if err != nil {
		return nil, err
	}

	defer buf.Reset()

	return buf.Bytes(), nil
}

func (d CSVDecoder) StreamDecode(r io.Reader) func(any) error {
	reader := NewCSVReader(r, d.opts...)
	return func(v any) error {
		return readCSVRecord(reader, v)
	}
}

func (d CSVDecoder) Decode(b []byte, v any) error {
	buf, reader := d.getReader()

	_, err := buf.Write(b)
	if err != nil {
		return err
	}

	defer buf.Reset()

	return readCSVRecord(reader, v)
}

func writeCSVRecord(writer *csv.Writer, v any) error {
	record, err := toCSVRecord(v)
	if err != nil {
		return err
	}

	err = writer.Write(record)
	if err != nil {
		return err
	}

	writer.Flush()

	return writer.Error()
}

func readCSVRecord(reader *csv.Reader, v any) error {
	record, err := reader.Read()
	if err != nil {
		return err
	}

	return fromCSVRecord(record, v)
}

func fromCSVRecord(record []string, v any) (err error) {
	switch v := v.(type) {
	case *[]string:
		*v = record
	case *[]any:
		*v = toAnySlice(record)
	case *any:
		*v = any(toAnySlice(record))
	case *structpb.Value:
		if v != nil {
			list, _ := structpb.NewList(toAnySlice(record))
			v.Kind = &structpb.Value_ListValue{
				ListValue: list,
			}
		}
	case *structpb.ListValue:
		if v != nil {
			list, _ := structpb.NewList(toAnySlice(record))
			v.Values = list.Values
		}
	case json.RawMessage:
	}
	return
}

func toCSVRecord(v any) (_ []string, err error) {
	switch v := v.(type) {
	case []string:
		return v, nil
	case []any:
		rec := make([]string, len(v))
		for idx, v := range v {
			if v == nil {
				rec[idx] = ""
			} else {
				rec[idx] = fmt.Sprint(v)
			}
		}
		return rec, nil
	case *structpb.Value:
		return toCSVRecord(v.AsInterface())
	case *structpb.ListValue:
		return toCSVRecord(v.AsSlice())
		// case *timestamppb.Timestamp:
		// 	return []string{v.AsTime().String()}, nil
		// case *durationpb.Duration:
		// 	return []string{v.AsDuration().String()}, nil
		// case proto.Message:
		// 	msg := v.ProtoReflect()
		// 	rec := make([]string, msg.Descriptor().Fields().Len())
		// 	for idx := range msg.Descriptor().Fields().Len() {
		// 		field := msg.Descriptor().Fields().Get(idx)
		// 		if msg.Has(field) {
		// 			if field.Message() != nil {
		// 				b, err := defaultJsonEncoder.Encode(msg.Get(field).Message().Interface())
		// 				if err != nil {
		// 					return nil, err
		// 				}
		// 				rec[idx] = string(b)
		// 			} else {
		// 				rec[idx] = msg.Get(field).String()
		// 			}
		// 		} else {
		// 			rec[idx] = ""
		// 		}
		// 	}
		// 	return rec, nil
	}

	b, err := defaultJsonEncoder.Encode(v)
	if err != nil {
		return nil, err
	}

	var jsonSlice []json.RawMessage
	err = defaultJsonDecoder.Decode(b, &jsonSlice)
	if err != nil {
		return nil, err
	}

	r := make([]string, len(jsonSlice))
	for idx, v := range jsonSlice {
		switch v.Kind() {
		case jsontext.Null.Kind():
			r[idx] = ""
		case jsontext.KindString:
			r[idx], err = strconv.Unquote(string(v))
			if err != nil {
				return nil, err
			}
		default:
			r[idx] = v.String()
		}
	}

	return r, nil
}

func toAnySlice[T any](from []T) (to []any) {
	if len(from) > 0 {
		to = make([]any, len(from))
		for idx, val := range from {
			to[idx] = any(val)
		}
	}
	return
}
