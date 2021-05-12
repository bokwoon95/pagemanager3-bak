package hy

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type textValue struct {
	values []interface{}
	unsafe bool
}

func (txt textValue) WriteHTML(buf *bytes.Buffer, _ Sanitizer) error {
	last := len(txt.values) - 1
	for i, value := range txt.values {
		switch value := value.(type) {
		case string:
			if value == "" {
				continue
			}
			if txt.unsafe {
				buf.WriteString(value)
			} else {
				escapeHTML(buf, htmlReplacementTable, value)
			}
			if strings.TrimSpace(value) == "" {
				continue
			}
			r, _ := utf8.DecodeLastRuneInString(value)
			if i != last && !unicode.IsSpace(r) {
				buf.WriteByte(' ')
			}
		default:
			if txt.unsafe {
				buf.WriteString(stringify(value))
			} else {
				escapeHTML(buf, htmlReplacementTable, stringify(value))
			}
		}
	}
	return nil
}

func Txt(a ...interface{}) Element {
	return textValue{values: a}
}

func UnsafeTxt(a ...interface{}) Element {
	return textValue{values: a, unsafe: true}
}

// adapted from database/sql:asString, text/template:printableValue,printValue
func stringify(v interface{}) string {
	switch v := v.(type) {
	case fmt.Stringer:
		return v.String()
	case string:
		return v
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.RFC3339Nano)
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'g', -1, 64)
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return "[nil]"
	}
	rv := reflect.ValueOf(v)
	for {
		if rv.Kind() != reflect.Ptr && rv.Kind() != reflect.Interface {
			break
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return "[no value]"
	}
	if rv.Kind() == reflect.Chan {
		return "[channel]"
	}
	if rv.Kind() == reflect.Func {
		return "[function]"
	}
	return fmt.Sprint(v)
}
