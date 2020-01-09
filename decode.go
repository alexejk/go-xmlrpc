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
	errFormatInvalidFieldType = "invalid field type: expected '%s', got '%s'"
)

// Decoder implementations provide mechanisms for parsing of XML-RPC responses to native data-types.
type Decoder interface {
	DecodeRaw(body []byte, v interface{}) error
	Decode(response *Response, v interface{}) error
	DecodeFault(response *Response) *Fault
}

// StdDecoder is the default implementation of the Decoder interface.
type StdDecoder struct{}

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

		if err := d.decodeValue(&param.Value, &field); err != nil {
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
			if m.Value.Int != "" {
				f.Code, _ = strconv.Atoi(m.Value.Int)
			} else {
				f.Code, _ = strconv.Atoi(m.Value.Int4)
			}
		case "faultString":
			f.String = m.Value.String
		}
	}

	return f
}

func (d *StdDecoder) decodeValue(value *ResponseValue, field *reflect.Value) error {

	var val interface{}
	var err error

	switch {

	case value.Int != "":
		val, err = strconv.Atoi(value.Int)

	case value.Int4 != "":
		val, err = strconv.Atoi(value.Int4)

	case value.Double != "":
		val, err = strconv.ParseFloat(value.Double, 64)

	case value.Boolean != "":
		val, err = d.decodeBoolean(value.Boolean)

	case value.String != "":
		val, err = value.String, nil

	case value.Base64 != "":
		val, err = d.decodeBase64(value.Base64)

	case value.DateTime != "":
		val, err = d.decodeDateTime(value.DateTime)

	// Array decoding
	case len(value.Array) > 0:

		if field.Kind() != reflect.Slice {
			return fmt.Errorf(errFormatInvalidFieldType, reflect.Slice.String(), field.Kind().String())
		}

		slice := reflect.MakeSlice(reflect.TypeOf(field.Interface()), len(value.Array), len(value.Array))
		for i, v := range value.Array {
			item := slice.Index(i)
			if err := d.decodeValue(v, &item); err != nil {
				return fmt.Errorf("failed decoding array item at index %d: %w", i, err)
			}
		}

		val = slice.Interface()

	// Struct decoding
	case len(value.Struct) != 0:

		// TODO: Support following *Ptr
		if field.Kind() != reflect.Struct {
			return fmt.Errorf(errFormatInvalidFieldType, reflect.Struct.String(), field.Kind().String())
		}

		for _, m := range value.Struct {

			// Upper-case the name
			fName := structMemberToFieldName(m.Name)
			f := field.FieldByName(fName)

			if !f.IsValid() {
				return fmt.Errorf("cannot find field '%s' on struct", fName)
			}

			if err := d.decodeValue(&m.Value, &f); err != nil {
				return fmt.Errorf("failed decoding struct member '%s': %w", m.Name, err)
			}
		}

	default:
		// NADA
	}

	if err != nil {
		return err
	}

	if val != nil {
		field.Set(reflect.ValueOf(val))
	}

	return nil
}

func (d *StdDecoder) decodeBoolean(value string) (bool, error) {

	switch value {
	case "1", "true", "TRUE", "True":
		return true, nil
	case "0", "false", "FALSE", "False":
		return false, nil
	}
	return false, fmt.Errorf("unrecognized value '%s' for boolean", value)
}

func (d *StdDecoder) decodeBase64(value string) ([]byte, error) {

	return base64.StdEncoding.DecodeString(value)
}

func (d *StdDecoder) decodeDateTime(value string) (time.Time, error) {

	return time.Parse(time.RFC3339, value)
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
