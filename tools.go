package gptease

import (
	"encoding/json"
	"reflect"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type Tool struct {
	Name        string
	Description string
	Parameters  string
	Handler     func(input string) (output string, err error)
}

func (t *Tool) openaiTool() openai.Tool {
	return openai.Tool{
		Type: "function",
		Function: openai.FunctionDefinition{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  json.RawMessage(t.Parameters),
		},
	}
}

type fieldSpec struct {
	Type        string               `json:"type"`
	Properties  map[string]fieldSpec `json:"properties,omitempty"`
	Items       *fieldSpec           `json:"items,omitempty"`
	Description string               `json:"description,omitempty"`
	Required    []string             `json:"required,omitempty"`
	Enum        []string             `json:"enum,omitempty"`
}

type spec struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Parameters  fieldSpec `json:"parameters"`
}

func (s *fieldSpec) parseTag(tag reflect.StructTag) {
	if d, ok := tag.Lookup("desc"); ok {
		s.Description = d
	}
	if e, ok := tag.Lookup("enum"); ok {
		s.Enum = strings.Split(e, ",")
	}
}

func readSpec(t reflect.Type) (s fieldSpec) {
	switch t.Kind() {
	case reflect.Struct:
		s.Type = "object"
		s.Properties = make(map[string]fieldSpec)
		for i := 0; i < t.NumField(); i++ {
			var f = t.Field(i)
			var name = f.Name
			// If the field has a JSON tag, use that as the property name.
			if jt := f.Tag.Get("json"); jt != "" {
				name = strings.Split(jt, ",")[0]
				if !strings.Contains(jt, "omitempty") {
					s.Required = append(s.Required, name)
				}
			}
			var fs = readSpec(f.Type)
			fs.parseTag(f.Tag)
			s.Properties[name] = fs
		}
	case reflect.Slice:
		s.Type = "array"
		var itemSpec = readSpec(t.Elem())
		s.Items = &itemSpec
	case reflect.String:
		s.Type = "string"
	case reflect.Int:
		s.Type = "integer"
	case reflect.Float32:
		s.Type = "number"
	case reflect.Bool:
		s.Type = "boolean"
	default:
		panic("unsupported type")
	}
	return s
}

// MakeTool generates a Tool definition from a function, by examining its
// signature and analysing the argument type using reflection.
//
// The function must be of the form:
//
//	func(arg arg) (ret ret, err error)
//
// In case the argument type is a struct, its fields can be annotated with
// field tags. A "json" tag will be used to determine the name of the field
// and whether it is required. A "desc" tag can be used to provide a
// description of the field. An "enum" tag can be used to provide a list of
// possible values for the field.
//
// Example of an argument struct with field tags:
//
//	type args struct {
//		Fruit         string `json:"text" desc:"your favourite fruit" enum:"apple,banana,orange"`
//		Consumption   []int  `json:"consumption,omitempty" desc:"number of fruits eaten each day"`
//	}
func MakeTool(f any, name, desc string) Tool {
	var t = reflect.TypeOf(f)
	// These are basically a compile-time errors. It should never depend on
	// the input, so it's perfectly appropriate to panic.
	switch {
	case t.Kind() != reflect.Func:
		panic("not a function")
	case t.NumIn() != 1:
		panic("not a function of one argument")
	case t.NumOut() != 2:
		panic("not a function of two results")
	case t.Out(1).Kind() != reflect.Interface:
		panic("second result is not an error")
	case !t.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()):
		panic("second result is not an error")
	}

	var params = readSpec(t.In(0))

	var b, err = json.MarshalIndent(params, "", "  ")
	if err != nil {
		panic(err)
	}

	return Tool{
		Name:        name,
		Description: desc,
		Parameters:  string(b),
		Handler: func(input string) (output string, err error) {
			var v = reflect.New(t.In(0))
			if err := json.Unmarshal([]byte(input), v.Interface()); err != nil {
				return "", err
			}
			var results = reflect.ValueOf(f).Call([]reflect.Value{v.Elem()})
			if !results[1].IsNil() {
				return "", results[1].Interface().(error)
			}
			var b, jerr = json.MarshalIndent(results[0].Interface(), "", "  ")
			if err != nil {
				panic(jerr)
			}
			return string(b), nil
		},
	}
}
