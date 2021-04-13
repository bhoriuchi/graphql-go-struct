package gqlstruct

import (
	"encoding/json"
	"strings"

	tools "github.com/bhoriuchi/graphql-go-tools"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
)

var primitiveMap = map[string]string{
	"Int":     "int64",
	"Float":   "float64",
	"String":  "string",
	"Boolean": "bool",
}

var protoMap = map[string]string{
	"Int":     "int64",
	"Float":   "float",
	"String":  "string",
	"Boolean": "bool",
}

var nullableProtoMap = map[string]string{
	"Int":     "google.protobuf.Int64Value",
	"Float":   "google.protobuf.DoubleValue",
	"String":  "google.protobuf.StringValue",
	"Boolean": "google.protobuf.BoolValue",
}

func getGoType(registry *Registry, name string) string {
	if primitive, found := primitiveMap[name]; found {
		return primitive
	}

	if def, found := registry.Structs[name]; found {
		return def.Name
	}

	return ""
}

func getPrototype(registry *Registry, ft *FieldType) string {
	baseName := ""
	if prototype, found := protoMap[ft.gqltype]; found {
		baseName = prototype
	} else if def, found := registry.Structs[ft.gqltype]; found {
		baseName = def.Name
	}

	if baseName != "" {
		parts := []string{}
		if ft.isList {
			parts = append(parts, "repeated")
		}
		if !ft.isNonNull {
			if wrapper, found := nullableProtoMap[ft.gqltype]; found {
				parts = append(parts, wrapper)
			}
		} else {
			parts = append(parts, baseName)
		}

		return strings.Join(parts, " ")
	}

	return ""
}

func getFieldType(registry *Registry, t ast.Type, ft *FieldType) *FieldType {
	if ft == nil {
		ft = &FieldType{}
	}

	switch t.GetKind() {
	case kinds.List:
		ft.isList = true
		return getFieldType(registry, t.(*ast.List).Type, ft)
	case kinds.NonNull:
		ft.isNonNull = true
		return getFieldType(registry, t.(*ast.NonNull).Type, ft)
	case kinds.Named:
		ft.gqltype = t.(*ast.Named).Name.Value
		ft.gotype = getGoType(registry, ft.gqltype)
		ft.prototype = getPrototype(registry, ft)
	}

	return ft
}

func getDirectiveConfig(directives []*ast.Directive) (*Args, error) {
	for _, dir := range directives {
		if dir.Name.Value == structDirectiveName {
			args, err := tools.GetArgumentValues(
				StructDirective.Args,
				dir.Arguments,
				map[string]interface{}{},
			)

			if err != nil {
				return &Args{}, err
			}

			c := &Args{}
			if err := mapProto(args, c); err != nil {
				return &Args{}, err
			}

			return c, nil
		}
	}

	return &Args{}, nil
}

func mapProto(in, out interface{}) error {
	j, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(j, out)
}

func titleCase(value string) string {
	name := strings.ReplaceAll(value, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.Title(name)
	return strings.ReplaceAll(name, " ", "")
}
