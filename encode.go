package xmlrpc

import (
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Encoder implementations are responsible for handling encoding of XML-RPC requests to the proper wire format.
type Encoder interface {
	Encode(w io.Writer, methodName string, args interface{}) error
}

// StdEncoder is the default implementation of Encoder interface.
type StdEncoder struct {
	// timeFormatter is optional - defaultTimeFormatter is used when unset.
	timeFormatter TimeFormatter
}

func (e *StdEncoder) Encode(w io.Writer, methodName string, args interface{}) error {
	x := newXMLWriter(w)

	x.raw("<methodCall>")
	x.element("methodName", methodName)

	if args != nil {
		if err := e.encodeArgs(x, args); err != nil {
			// A failed write is the root cause and outranks whatever it caused downstream
			if x.err != nil {
				return x.err
			}

			return fmt.Errorf("cannot encoded provided method arguments: %w", err)
		}
	}

	x.raw("</methodCall>")

	return x.err
}

func (e *StdEncoder) encodeArgs(x *xmlWriter, args interface{}) error {
	// Allows reading both pointer and value-structs
	elem := reflect.Indirect(reflect.ValueOf(args))

	switch elem.Kind() {
	case reflect.Map:
		return e.encodeBareMapArgs(x, elem)
	case reflect.Struct:
		return e.encodeStructArgs(x, elem)
	default:
		return fmt.Errorf("unsupported argument type %s - use stuct{} wrapper with exported fields (or map[string]{} if single <struct> param is expected) ", elem.Kind().String())
	}
}

func (e *StdEncoder) encodeStructArgs(x *xmlWriter, elem reflect.Value) error {
	numFields := elem.NumField()
	if numFields == 0 {
		return nil
	}

	hasExportedFields := false
	for fN := 0; fN < numFields; fN++ {
		field := elem.Field(fN)
		if !field.CanInterface() {
			continue
		}

		// If this is first exported field - print out <params> tag
		if !hasExportedFields {
			hasExportedFields = true
			x.raw("<params>")
		}

		x.raw("<param>")
		if err := e.encodeValue(x, field.Interface()); err != nil {
			return fmt.Errorf("cannot encode argument '%s': %w", elem.Type().Field(fN).Name, err)
		}
		x.raw("</param>")
	}

	// Only write closing </params> tag if at least one exported field is found
	if hasExportedFields {
		x.raw("</params>")
	}

	return nil
}

func (e *StdEncoder) encodeBareMapArgs(x *xmlWriter, elem reflect.Value) error {
	if elem.Type().Key().Kind() != reflect.String {
		return fmt.Errorf("unsupported type %s for bare map key, only string keys are supported", elem.Type().Key().Kind().String())
	}

	x.raw("<params><param>")
	if err := e.encodeValue(x, elem.Interface()); err != nil {
		return fmt.Errorf("cannot encode bare map argument: %w", err)
	}
	x.raw("</param></params>")

	return nil
}

// encodeValue will encode input into the XML-RPC compatible format.
// If provided value is a pointer, value of pointer will be used, unless pointer is nil.
// In that case a <nil/> value is returned.
//
// See more: https://en.wikipedia.org/wiki/XML-RPC#Data_types
func (e *StdEncoder) encodeValue(x *xmlWriter, value interface{}) error {
	// Stop traversing as soon as the writer has failed - the remaining output is discarded
	if x.err != nil {
		return x.err
	}

	valueOf := reflect.ValueOf(value)
	kind := valueOf.Kind()

	// Handling pointers by following them.
	if kind == reflect.Pointer {
		if valueOf.IsNil() {
			x.raw("<value><nil/></value>")

			return nil
		}

		return e.encodeValue(x, valueOf.Elem().Interface())
	}

	x.raw("<value>")
	switch kind {
	case reflect.Bool:
		e.encodeBoolean(x, value.(bool))

	case reflect.Int:
		e.encodeInteger(x, value.(int))

	case reflect.Float64:
		if err := e.encodeDouble(x, value.(float64)); err != nil {
			return fmt.Errorf("cannot encode double value: %w", err)
		}

	case reflect.String:
		e.encodeString(x, value.(string))

	case reflect.Array, reflect.Slice:

		if e.isByteArray(value) {
			e.encodeBase64(x, value.([]byte))
		} else {
			if err := e.encodeArray(x, value); err != nil {
				return fmt.Errorf("cannot encode array value: %w", err)
			}
		}

	case reflect.Struct:
		if timeValue, ok := value.(time.Time); ok {
			e.encodeTime(x, timeValue)
		} else {
			if err := e.encodeStruct(x, value); err != nil {
				return fmt.Errorf("cannot encode struct value: %w", err)
			}
		}

	case reflect.Map:
		if err := e.encodeMap(x, value); err != nil {
			return fmt.Errorf("cannot encode map value: %w", err)
		}

	default:
		return fmt.Errorf("unsupported type %v", kind)
	}

	x.raw("</value>")

	return nil
}

func (e *StdEncoder) isByteArray(val interface{}) bool {
	_, ok := val.([]byte)

	return ok
}

func (e *StdEncoder) encodeInteger(x *xmlWriter, val int) {
	x.element("int", strconv.Itoa(val))
}

func (e *StdEncoder) encodeDouble(x *xmlWriter, val float64) error {
	// XML-RPC has no representation for these, and emitting them produces a document the
	// receiver cannot interpret as a number
	if math.IsNaN(val) {
		return fmt.Errorf("unsupported value NaN")
	}

	if math.IsInf(val, 1) {
		return fmt.Errorf("unsupported value +Inf")
	}

	if math.IsInf(val, -1) {
		return fmt.Errorf("unsupported value -Inf")
	}

	// The specification allows only decimal point notation - no exponent - so 'f' is used
	// with a precision of -1, giving the fewest digits that still round-trip exactly
	formatted := strconv.FormatFloat(val, 'f', -1, 64)

	// A whole number formats without a period, which the grammar does not allow
	if !strings.ContainsRune(formatted, '.') {
		formatted += ".0"
	}

	x.element("double", formatted)

	return nil
}

func (e *StdEncoder) encodeBoolean(x *xmlWriter, val bool) {
	v := "0"
	if val {
		v = "1"
	}

	x.element("boolean", v)
}

func (e *StdEncoder) encodeString(x *xmlWriter, val string) {
	x.element("string", val)
}

func (e *StdEncoder) encodeArray(x *xmlWriter, val interface{}) error {
	valueOf := reflect.ValueOf(val)

	x.raw("<array><data>")
	for i := 0; i < valueOf.Len(); i++ {
		if err := e.encodeValue(x, valueOf.Index(i).Interface()); err != nil {
			return fmt.Errorf("cannot encode array element at index %d: %w", i, err)
		}
	}
	x.raw("</data></array>")

	return nil
}

func (e *StdEncoder) encodeStruct(x *xmlWriter, val interface{}) error {
	typeOf := reflect.TypeOf(val)
	valueOf := reflect.ValueOf(val)

	x.raw("<struct>")
	for i := 0; i < typeOf.NumField(); i++ {
		field := valueOf.Field(i)
		// Skip over unexported fields
		if !field.CanInterface() {
			continue
		}

		fieldType := typeOf.Field(i)
		fieldName := fieldType.Name
		tag := fieldType.Tag
		if tag.Get("xml") != "" {
			fieldName = tag.Get("xml")
		} else if tag.Get("xmlrpc") != "" {
			fieldName = tag.Get("xmlrpc")
		}

		x.raw("<member>")
		x.element("name", fieldName)

		if err := e.encodeValue(x, field.Interface()); err != nil {
			return fmt.Errorf("cannot encode value of struct field '%s': %w", fieldName, err)
		}
		x.raw("</member>")
	}
	x.raw("</struct>")

	return nil
}

func (e *StdEncoder) encodeBase64(x *xmlWriter, val []byte) {
	x.element("base64", base64.StdEncoding.EncodeToString(val))
}

func (e *StdEncoder) encodeTime(x *xmlWriter, val time.Time) {
	x.element("dateTime.iso8601", timeFormatterOrDefault(e.timeFormatter).FormatTime(val))
}

func (e *StdEncoder) encodeMap(x *xmlWriter, val interface{}) error {
	mapValue := reflect.ValueOf(val)
	iter := mapValue.MapRange()

	x.raw("<struct>")
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Convert key to string
		keyStr := fmt.Sprintf("%v", key.Interface())

		x.raw("<member>")
		x.element("name", keyStr)

		if err := e.encodeValue(x, value.Interface()); err != nil {
			return fmt.Errorf("cannot encode map value for key '%s': %w", keyStr, err)
		}
		x.raw("</member>")
	}
	x.raw("</struct>")

	return nil
}
