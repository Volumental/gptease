package gptease_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/Volumental/gptease"
)

func jsonEquals(a, b string) bool {
	if a == b {
		return true // Also handles the empty string.
	}
	var aa, bb interface{}
	if err := json.Unmarshal([]byte(a), &aa); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bb); err != nil {
		return false
	}
	return reflect.DeepEqual(aa, bb)
}

func TestGenerateTool(t *testing.T) {
	trivial := func(struct{}) (struct{}, error) { return struct{}{}, nil }

	trivialError := func(struct{}) (struct{}, error) { return struct{}{}, errors.New("trivial error") }

	primitive := func(i int) (int, error) { return i, nil }

	type args1 struct {
		Foo string `json:"foo"`
		Bar int    `json:"bar,omitempty"`
	}

	type ret1 struct {
		Baz string `json:"baz"`
	}

	func1 := func(args args1) (ret ret1, err error) {
		ret.Baz = args.Foo + fmt.Sprint(args.Bar)
		return
	}

	type args2 struct {
		List   []string `json:"list"`
		Nested struct {
			Qux string `json:"qux" enum:"foo,baar,baaaz" desc:"extra string"`
		} `json:"nested,omitempty"`
	}

	type inner struct {
		Num float32 `json:"num" desc:"length of a string"`
	}

	type ret2 struct {
		List []inner `json:"list" desc:"list of lengths"`
	}

	func2 := func(args args2) (ret ret2, err error) {
		ret.List = make([]inner, len(args.List))
		for i, s := range args.List {
			ret.List[i].Num = float32(len(s))
		}
		ret.List = append(ret.List, inner{Num: float32(len(args.Nested.Qux))})
		return
	}

	tests := []struct {
		name       string
		f          any
		desc       string
		wantName   string
		wantDesc   string
		wantParams string
		input      string
		wantOutput string
		wantError  bool
	}{
		{
			name:     "trivial",
			f:        trivial,
			desc:     "Trivial tool.",
			wantName: "trivial",
			wantDesc: "Trivial tool.",
			wantParams: `{
				"type": "object"
			}`,
			input:      "{}",
			wantOutput: "{}",
		},
		{
			name:     "trivialError",
			f:        trivialError,
			desc:     "Trivial tool with error.",
			wantName: "trivialError",
			wantDesc: "Trivial tool with error.",
			wantParams: `{
				"type": "object"
			}`,
			input:     "{}",
			wantError: true,
		},
		{
			name:     "primitive",
			f:        primitive,
			desc:     "Function taking and returning a primitive.",
			wantName: "primitive",
			wantDesc: "Function taking and returning a primitive.",
			wantParams: `{
				"type": "integer"
			}`,
			input:      "42",
			wantOutput: "42",
		},
		{
			name:     "simpleStructs",
			f:        func1,
			desc:     "Function taking and returning simple structs.",
			wantName: "simpleStructs",
			wantDesc: "Function taking and returning simple structs.",
			wantParams: `{
				"type": "object",
				"properties": {
					"foo": {
						"type": "string"
					},
					"bar": {
						"type": "integer"
					}
				},
				"required": ["foo"]
			}`,
			input:      `{"foo": "hello", "bar": 42}`,
			wantOutput: `{"baz": "hello42"}`,
		},
		{
			name:     "complexStructs",
			f:        func2,
			desc:     "Function taking and returning complex structs.",
			wantName: "complexStructs",
			wantDesc: "Function taking and returning complex structs.",
			wantParams: `{
				"type": "object",
				"properties": {
					"list": {
						"type": "array",
						"items": {
							"type": "string"
						}
					},
					"nested": {
						"type": "object",
						"properties": {
							"qux": {
								"type": "string",
								"enum": ["foo", "baar", "baaaz"],
								"description": "extra string"
							}
						},
						"required": ["qux"]
					}
				},
				"required": ["list"]
			}`,
			input:      `{"list": ["hello", "world"], "nested": {"qux": "foo"}}`,
			wantOutput: `{"list": [{"num": 5}, {"num": 5}, {"num": 3}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tool = gptease.MakeTool(tt.f, tt.name, tt.desc)
			if tool.Name != tt.wantName {
				t.Errorf("GenerateTool().Name = %v, want %v", tool.Name, tt.wantName)
			}
			if tool.Description != tt.wantDesc {
				t.Errorf("GenerateTool().Description = %v, want %v", tool.Description, tt.wantDesc)
			}
			if !jsonEquals(tool.Parameters, tt.wantParams) {
				t.Errorf("GenerateTool().Spec = %v, want %v", tool.Parameters, tt.wantParams)
			}

			if got, err := tool.Handler(tt.input); (err != nil) != tt.wantError {
				t.Errorf("Handler error = %v, want %v", err, tt.wantError)
			} else if !jsonEquals(got, tt.wantOutput) {
				t.Errorf("Handler output = %v, want %v", got, tt.wantOutput)
			}
		})
	}
}
