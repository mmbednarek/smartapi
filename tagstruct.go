package smartapi

import (
	"errors"
	"reflect"
	"strings"
)

func parseArgument(tag string, field reflect.StructField) (Argument, error) {
	var kind string
	var data string
	eqAt := strings.Index(tag, "=")
	if eqAt >= 0 {
		kind = tag[:eqAt]
		data = tag[(eqAt+1):]
	} else {
		kind = tag
	}
	a, err := getArgument(kind, data, field)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func getArgument(kind string, data string, field reflect.StructField) (Argument, error) {
	switch kind {
	case "header":
		return headerArgument{name: data}, nil
	case "r_header":
		return requiredHeaderArgument{name: data}, nil
	case "json_body":
		return jsonBodyDirectArgument{typ: field.Type}, nil
	case "string_body":
		return stringBodyArgument{}, nil
	case "byte_slice_body":
		return byteSliceBodyArgument{}, nil
	case "body_reader":
		return bodyReaderArgument{}, nil
	case "url_param":
		return urlParamArgument{name: data}, nil
	case "context":
		return contextArgument{}, nil
	case "query_param":
		return queryParamArgument{name: data}, nil
	case "post_query_param":
		return postQueryParamArgument{name: data}, nil
	case "cookie":
		return cookieArgument{name: data}, nil
	case "response_headers":
		return headerSetterArgument{}, nil
	case "response_cookies":
		return cookieSetterArgument{}, nil
	case "response_writer":
		return responseWriterArgument{}, nil
	case "request":
		return fullRequestArgument{}, nil
	case "request_struct":
		if field.Type.Kind() != reflect.Ptr {
			if field.Type.Kind() != reflect.Struct {
				return nil, errors.New("invalid type of request_struct")
			}
			s, err := requestStruct(field.Type)
			if err != nil {
				return nil, err
			}
			return tagStructDirectArgument{
				structType: s.structType,
				flags:      s.flags,
				arguments:  s.arguments,
			}, nil
		}
		return requestStruct(field.Type.Elem())
	}
	return nil, errors.New("unsupported tag")
}
