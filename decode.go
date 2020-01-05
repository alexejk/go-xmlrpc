package xmlrpc

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

type respWrapper struct {
	Params []respParam `xml:"params>param"`
	Fault  *respFault  `xml:"fault,omitempty"`
}

type respParam struct {
	Value respValue `xml:"value"`
}

type respValue struct {
	Array    []*respValue        `xml:"array>data>value"`
	Struct   []*respStructMember `xml:"struct>member"`
	String   string              `xml:"string"`
	Int      string              `xml:"int"`
	Int4     string              `xml:"i4"`
	Double   string              `xml:"double"`
	Boolean  string              `xml:"boolean"`
	DateTime string              `xml:"dateTime.iso8601"`
	Base64   string              `xml:"base64"`

	Raw string `xml:",innerxml"` // the value can be default string
}

type respStructMember struct {
	Name  string    `xml:"name"`
	Value respValue `xml:"value"`
}

type respFault struct {
	Value respValue `xml:"value"`
}

func (f respFault) IsEmpty() bool {
	return len(f.Value.Struct) == 0
}

func DecodeResponse(body string, v interface{}) error {

	wrapper := &respWrapper{}
	if err := xml.Unmarshal([]byte(body), wrapper); err != nil {
		return err
	}

	if wrapper.Fault != nil {
		// TODO: Handle fault response
	}

	// Validate that v has same number of public fields as response params
	if err := fieldsMustEqual(v, len(wrapper.Params)); err != nil {
		return err
	}

	vElem := reflect.Indirect(reflect.ValueOf(v))
	for i, param := range wrapper.Params {
		field := vElem.Field(i)

		if err := decodeValue(&param.Value, &field); err != nil {
			return err
		}
	}

	return nil
}

func decodeValue(value *respValue, field *reflect.Value) error {

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
		val, err = decodeBoolean(value.Boolean)

	case value.String != "":
		val, err = value.String, nil

	case value.Base64 != "":
		val, err = decodeBase64(value.Base64)

	case value.DateTime != "":
		val, err = decodeDateTime(value.DateTime)

	case len(value.Array) > 0:
		slice := reflect.MakeSlice(reflect.TypeOf(field.Interface()), len(value.Array), len(value.Array))
		for i, v := range value.Array {
			item := slice.Index(i)
			if err := decodeValue(v, &item); err != nil {
				return fmt.Errorf("failed decoding array item at index %d: %w", i, err)
			}
		}

		val = slice.Interface()
	}

	if err != nil {
		return err
	}

	if val != nil {
		field.Set(reflect.ValueOf(val))
	}

	return nil
}

func decodeBoolean(value string) (bool, error) {

	switch value {
	case "1", "true", "TRUE", "True":
		return true, nil
	case "0", "false", "FALSE", "False":
		return false, nil
	}
	return false, fmt.Errorf("unrecognized value '%s' for boolean", value)
}

func decodeBase64(value string) ([]byte, error) {

	return base64.StdEncoding.DecodeString(value)
}

func decodeDateTime(value string) (time.Time, error) {

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
