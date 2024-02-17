package xmlrpc

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	errFormatInvalidFieldType           = "invalid field type: expected '%s', got '%s'"
	errFormatInvalidFieldTypeOrType     = "invalid field type: expected '%s' or '%s', got '%s'"
	errFormatInvalidMapKeyTypeForStruct = "invalid map key type: must be 'string' when decoding structs into a map, got '%s'"
	float64BitSize                      = 64
)

// Decoder implementations provide mechanisms for parsing of XML-RPC responses to native data-types.
type Decoder interface {
	DecodeRaw(body []byte, v interface{}) error
	Decode(response *Response, v interface{}) error
	DecodeFault(response *Response) *Fault
}

// StdDecoder is the default implementation of the Decoder interface.
type StdDecoder struct {
	skipUnknownFields bool
}

func (d *StdDecoder) DecodeRaw(body []byte, v interface{}) error {
	response, err := NewResponse(body)
	if err != nil {
		return err
	}

	if response.Fault != nil {
		return d.decodeFault(response.Fault)
	}

	return d.Decode(response, v)
}

func (d *StdDecoder) Decode(response *Response, v interface{}) error {
	// Validate that v has same number of public fields as response params
	if err := fieldsMustEqual(v, len(response.Params)); err != nil {
		return err
	}

	vElem := reflect.Indirect(reflect.ValueOf(v))
	for i, param := range response.Params {
		field := vElem.Field(i)

		if err := d.decodeValue(&param.Value, field); err != nil {
			return err
		}
	}

	return nil
}

func (d *StdDecoder) DecodeFault(response *Response) *Fault {
	if response.Fault == nil {
		return nil
	}

	return d.decodeFault(response.Fault)
}

func (d *StdDecoder) decodeFault(fault *ResponseFault) *Fault {
	f := &Fault{}
	for _, m := range fault.Value.Struct {
		switch m.Name {
		case "faultCode":
			if m.Value.Int != nil {
				f.Code, _ = strconv.Atoi(*m.Value.Int)
			} else if m.Value.Int4 != nil {
				f.Code, _ = strconv.Atoi(*m.Value.Int4)
			} else {
				f.Code = 0 // Unknown fault code
			}
		case "faultString":
			if m.Value.String != nil {
				f.String = *m.Value.String
			}
		}
	}

	return f
}

func (d *StdDecoder) decodeValue(value *ResponseValue, field reflect.Value) error {
	field = indirect(field)

	var val interface{}
	var err error

	switch {
	case value.Int != nil:
		val, err = d.decodeInt(*value.Int)

	case value.Int4 != nil:
		val, err = d.decodeInt(*value.Int4)

	case value.Double != nil:
		val, err = d.decodeDouble(*value.Double)

	case value.Boolean != nil:
		val, err = d.decodeBoolean(*value.Boolean)

	case value.String != nil:
		val, err = *value.String, nil

	case value.Base64 != nil:
		val, err = d.decodeBase64(*value.Base64)

	case value.DateTime != nil:
		val, err = d.decodeDateTime(*value.DateTime)

	// Array decoding
	case value.Array != nil:
		if field.Kind() != reflect.Slice {
			return fmt.Errorf(errFormatInvalidFieldType, reflect.Slice.String(), field.Kind().String())
		}

		values := value.Array.Values
		if len(values) == 0 {
			val, err = nil, nil
			break
		}

		slice := reflect.MakeSlice(reflect.TypeOf(field.Interface()), len(values), len(values))
		for i, v := range values {
			item := slice.Index(i)
			if err := d.decodeValue(v, item); err != nil {
				return fmt.Errorf("failed decoding array item at index %d: %w", i, err)
			}
		}

		val = slice.Interface()

	// Struct decoding
	case len(value.Struct) != 0:
		fieldKind := field.Kind()
		if fieldKind != reflect.Struct && fieldKind != reflect.Map {
			return fmt.Errorf(errFormatInvalidFieldTypeOrType, reflect.Struct.String(), reflect.Map.String(), fieldKind.String())
		}

		// If we are targeting a map, it should be initialized
		if fieldKind == reflect.Map {
			if kt := field.Type().Key().Kind(); kt != reflect.String {
				return fmt.Errorf(errFormatInvalidMapKeyTypeForStruct, kt.String())
			}

			if field.IsNil() {
				field.Set(reflect.MakeMap(field.Type()))
			}
		}

		for _, m := range value.Struct {
			if fieldKind == reflect.Map {
				mapKey := reflect.ValueOf(m.Name)
				f := reflect.New(field.Type().Elem()).Elem()

				if err := d.decodeValue(&m.Value, f); err != nil {
					return fmt.Errorf("failed decoding struct member '%s': %w", m.Name, err)
				}

				field.SetMapIndex(mapKey, f)
			} else {
				// Upper-case the name
				fName := structMemberToFieldName(m.Name)
				f := findFieldByNameOrTag(field, fName)

				if !f.IsValid() {
					if d.skipUnknownFields {
						continue
					}
					return fmt.Errorf("cannot find field '%s' on struct", fName)
				}

				if err := d.decodeValue(&m.Value, f); err != nil {
					return fmt.Errorf("failed decoding struct member '%s': %w", m.Name, err)
				}
			}
		}

	default:
		// Default to inner string (raw XML)
		val, err = value.RawXML, nil
	}

	if err != nil {
		return err
	}

	if val != nil {
		// Assign if directly assignable, or convert to type if convertible
		rVal := reflect.ValueOf(val)
		if rVal.Type().AssignableTo(field.Type()) {
			field.Set(rVal)
		} else {
			if !rVal.Type().ConvertibleTo(field.Type()) {
				return fmt.Errorf("type '%s' cannot be assigned a value of type '%s'", field.Type().String(), rVal.Type().String())
			}

			field.Set(rVal.Convert(field.Type()))
		}
	}

	return nil
}

func (d *StdDecoder) decodeInt(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func (d *StdDecoder) decodeDouble(value string) (float64, error) {
	if value == "" {
		return 0.0, nil
	}
	return strconv.ParseFloat(value, float64BitSize)
}

func (d *StdDecoder) decodeBoolean(value string) (bool, error) {
	switch value {
	case "1", "true", "TRUE", "True":
		return true, nil
	case "0", "false", "FALSE", "False":
		return false, nil
	case "":
		return false, nil
	default:
		return false, fmt.Errorf("unrecognized value '%s' for boolean", value)
	}
}

func (d *StdDecoder) decodeBase64(value string) ([]byte, error) {
	if value == "" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(value)
}

func (d *StdDecoder) decodeDateTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}

func findFieldByNameOrTag(field reflect.Value, fName string) reflect.Value {
	typ := field.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		keyName := getFieldNameFromTag(&f, "xmlrpc")

		// If tagged name matches search value - return
		if keyName == fName {
			return field.Field(i)
		}
	}

	return field.FieldByName(fName)
}

func getFieldNameFromTag(f *reflect.StructField, tagName string) string {
	var keyName string

	if f.PkgPath != "" {
		return keyName
	}

	tagValue := f.Tag.Get(tagName)
	if tagValue == "" {
		return keyName
	}

	// Determine the name of the field based on struct tag
	if index := strings.Index(tagValue, ","); index != -1 {
		if tagValue[:index] == "-" {
			return keyName
		}

		if keyNameTagValue := tagValue[:index]; keyNameTagValue != "" {
			keyName = keyNameTagValue
		}

		return keyName
	}
	if tagValue != "" && tagValue != "-" {
		keyName = tagValue
	}

	return keyName
}

func fieldsMustEqual(v interface{}, expectation int) error {
	vElem := reflect.Indirect(reflect.ValueOf(v))
	numFields := 0

	for i := 0; i < vElem.NumField(); i++ {
		if vElem.Field(i).CanInterface() {
			numFields++
		}
	}

	if numFields != expectation {
		return fmt.Errorf("number of exported fields (%d) on response type doesnt match expectation (%d)", numFields, expectation)
	}

	return nil
}

func structMemberToFieldName(structName string) string {
	b := new(strings.Builder)
	capNext := true

	for _, v := range structName {
		if v >= 'A' && v <= 'Z' {
			b.WriteRune(v)
		}
		if v >= '0' && v <= '9' {
			b.WriteRune(v)
		}

		if v >= 'a' && v <= 'z' {
			if capNext {
				b.WriteString(strings.ToUpper(string(v)))
			} else {
				b.WriteRune(v)
			}
		}

		if v == '_' || v == ' ' || v == '-' || v == '.' {
			capNext = true
		} else {
			capNext = false
		}
	}

	return b.String()
}

// indirect walks down v allocating pointers as needed,
// until it gets to a non-pointer.
//
// Adapted from encoding/json indirect() function
// https://golang.org/src/encoding/json/decode.go?#L480
func indirect(v reflect.Value) reflect.Value {
	// After the first round-trip, we set v back to the original value to
	// preserve the original RW flags contained in reflect.Value.
	v0 := v
	haveAddr := false

	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		haveAddr = true
		v = v.Addr()
	}

	for {
		// Load value from interface, but only if the result will be
		// usefully addressable.
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() {
				haveAddr = false
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		// Prevent infinite loop if v is an interface pointing to its own address:
		//     var v interface{}
		//     v = &v
		if v.Elem().Kind() == reflect.Interface && v.Elem().Elem() == v {
			v = v.Elem()
			break
		}

		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		if haveAddr {
			v = v0 // restore original value after round-trip Value.Addr().Elem()
			haveAddr = false
		} else {
			v = v.Elem()
		}
	}

	return v
}
