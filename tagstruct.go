package smartapi

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

func parseArgument(tag string, fieldType reflect.Type) (Argument, error) {
	var kind string
	var data string
	eqAt := strings.Index(tag, "=")
	if eqAt >= 0 {
		kind = tag[:eqAt]
		data = tag[(eqAt + 1):]
	} else {
		kind = tag
	}
	a, err := getArgument(kind, data, fieldType)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func getArgument(kind string, data string, fieldType reflect.Type) (Argument, error) {
	switch kind {
	case "header":
		return headerArgument{name: data}, nil
	case "r_header":
		return requiredHeaderArgument{name: data}, nil
	case "json_body":
		return jsonBodyDirectArgument{typ: fieldType}, nil
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
	case "r_query_param":
		return requiredQueryParamArgument{name: data}, nil
	case "post_query_param":
		return postQueryParamArgument{name: data}, nil
	case "r_post_query_param":
		return requiredPostQueryParamArgument{name: data}, nil
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
	case "as_int":
		arg, err := parseArgument(data, reflect.TypeOf(""))
		if err != nil {
			return nil, fmt.Errorf("(as int) %w", err)
		}
		asInt := AsInt(arg)
		if asInt.options().has(flagError) {
			return nil, fmt.Errorf("(as int) %w", asInt.(errorEndpointParam).err)
		}
		return asInt.(Argument), nil
	case "as_byte_slice":
		arg, err := parseArgument(data, reflect.TypeOf(""))
		if err != nil {
			return nil, fmt.Errorf("(as byte slice) %w", err)
		}
		asByteSlice := AsByteSlice(arg)
		if asByteSlice.options().has(flagError) {
			return nil, fmt.Errorf("(as byte slice) %w", asByteSlice.(errorEndpointParam).err)
		}
		return AsByteSlice(arg).(Argument), nil
	case "request_struct":
		if fieldType.Kind() != reflect.Ptr {
			if fieldType.Kind() != reflect.Struct {
				return nil, errors.New("invalid type of request_struct")
			}
			s, err := requestStruct(fieldType)
			if err != nil {
				return nil, err
			}
			return tagStructDirectArgument{
				structType: s.structType,
				flags:      s.flags,
				arguments:  s.arguments,
			}, nil
		}
		return requestStruct(fieldType.Elem())
	}
	return nil, errors.New("unsupported tag")
}
