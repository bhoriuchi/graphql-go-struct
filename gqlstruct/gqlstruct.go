package gqlstruct

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"text/template"

	tools "github.com/bhoriuchi/graphql-go-tools"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/iancoleman/strcase"
)

// Struct directive
const (
	structDirectiveName    = "struct"
	StructDirectiveTypeDef = `directive @struct (
	private: Boolean = false
	omit: Boolean = false
  name: String
	service: String
	type: String
	prototype: String
	tags: [String!]
) on FIELD_DEFINITION | OBJECT`
)

// registry key type
type registryKey string

// Actual registry key
var RegistryKey registryKey = "directive_struct_registry"

// Struct direcrtive
var StructDirective = graphql.NewDirective(graphql.DirectiveConfig{
	Name:        structDirectiveName,
	Description: "A directive to define struct types",
	Locations: []string{
		graphql.DirectiveLocationFieldDefinition,
		graphql.DirectiveLocationObject,
	},
	Args: graphql.FieldConfigArgument{
		"is_root": &graphql.ArgumentConfig{
			Type:         graphql.Boolean,
			DefaultValue: false,
		},
		"private": &graphql.ArgumentConfig{
			Type:         graphql.Boolean,
			DefaultValue: false,
		},
		"omit": &graphql.ArgumentConfig{
			Type:         graphql.Boolean,
			DefaultValue: false,
		},
		"name": &graphql.ArgumentConfig{
			Type: graphql.String,
		},
		"service": &graphql.ArgumentConfig{
			Type: graphql.String,
		},
		"type": &graphql.ArgumentConfig{
			Type: graphql.String,
		},
		"prototype": &graphql.ArgumentConfig{
			Type: graphql.String,
		},
		"tags": &graphql.ArgumentConfig{
			Type: graphql.NewList(
				graphql.NewNonNull(
					graphql.String,
				),
			),
		},
	},
})

// Args arguments
type Args struct {
	Private   bool     `json:"private"`
	Omit      bool     `json:"omit"`
	Name      string   `json:"name"`
	Service   string   `json:"service"`
	Type      string   `json:"type"`
	Prototype string   `json:"prototype"`
	Tags      []string `json:"tags"`
}

var StructDirectiveVisitor = &tools.SchemaDirectiveVisitor{
	VisitObject: func(p tools.VisitObjectParams) {
		def := tools.MergeExtensions(p.Node, p.Extensions...)
		// get the registry
		reg := p.Context.Value(RegistryKey)
		registry := reg.(*Registry)

		// map the arguments
		args := &Args{}
		if err := mapProto(p.Args, args); err != nil {
			fmt.Printf("Failed to map arguments: %v\n", err)
			return
		}

		// add the type if it does not exist
		name := p.Config.Name
		obj, found := registry.Structs[name]
		if !found {
			obj = &Def{
				Name:         name,
				Fields:       map[string]*FieldDef{},
				Private:      args.Private,
				ExplicitName: args.Name,
			}
			registry.Structs[name] = obj
		}

		// add the fields
		for _, field := range def.Fields {
			args, err := getDirectiveConfig(field.Directives)
			if err != nil {
				fmt.Printf("Failed to get field directive args: %v\n", err)
				return
			}

			// if the field is marked with a service, add the service and method
			if args.Service != "" {
				svc, found := registry.Services[args.Service]
				if !found {
					svc = &Service{
						Name:    args.Service,
						Methods: map[string]*ServiceMethod{},
					}
					registry.Services[args.Service] = svc
				}

				methodName := strcase.ToCamel(field.Name.Value)
				method, found := svc.Methods[methodName]
				if !found {
					method = &ServiceMethod{
						Name:     methodName,
						Response: field.Type.(*ast.Named).Name.Value,
					}
				}
				svc.Methods[methodName] = method
			}

			if args.Omit {
				continue
			}

			fieldName := field.Name.Value
			ft := getFieldType(registry, field.Type, nil)
			def := &FieldDef{
				Ref:          fieldName,
				Name:         fieldName,
				GqlType:      ft.gqltype,
				GoType:       ft.String(),
				Prototype:    ft.prototype,
				Private:      args.Private,
				ExplicitName: args.Name,
				Tags:         args.Tags,
			}

			if args.Type != "" {
				def.GoType = args.Type
			}

			if args.Prototype != "" {
				def.Prototype = args.Prototype
			}

			obj.Fields[fieldName] = def
		}
	},
}

// Make makes the types
func Make(config tools.ExecutableSchema) (context.Context, error) {
	registry := &Registry{
		RootQuery:        "",
		RootMutation:     "",
		RootSubscription: "",
		Services:         map[string]*Service{},
		Structs:          map[string]*Def{},
	}

	ctx := context.WithValue(context.Background(), RegistryKey, registry)

	if config.SchemaDirectives == nil {
		config.SchemaDirectives = tools.SchemaDirectiveVisitorMap{}
	}
	config.SchemaDirectives["struct"] = StructDirectiveVisitor

	if config.Resolvers == nil {
		config.Resolvers = map[string]interface{}{}
	}
	config.Resolvers["@struct"] = StructDirective

	schema, err := tools.MakeExecutableSchemaWithContext(ctx, config)
	if err != nil {
		return ctx, err
	}

	registry.RootQuery = schema.QueryType().Name()

	if schema.MutationType() != nil {
		registry.RootMutation = schema.MutationType().Name()
	}

	if schema.SubscriptionType() != nil {
		registry.RootSubscription = schema.SubscriptionType().Name()
	}

	for structName, s := range registry.Structs {
		if structName == registry.RootQuery || structName == registry.RootMutation || structName == registry.RootSubscription {
			s.IsRoot = true
		}

		for name, field := range s.Fields {
			if len(field.Tags) == 0 {
				field.Tags = []string{}
			}
			field.Tags = append(field.Tags, fmt.Sprintf(`json:"%s"`, name))
		}
	}

	funcMap := template.FuncMap{
		"serializeTags": func(tags []string) string {
			if len(tags) == 0 {
				return ""
			}

			return fmt.Sprintf("`%s`", strings.Join(tags, " "))
		},
		"title": titleCase,
		"objName": func(name, explicitName string, private bool) string {
			if explicitName != "" {
				return explicitName
			}

			if private {
				return strcase.ToLowerCamel(name)
			}

			return strcase.ToCamel(name)
		},
		"nextNumber": func(index int) int {
			return index + 1
		},
		"sortedFields": func(fields map[string]*FieldDef) []*FieldDef {
			var arr FieldDefs
			arr = []*FieldDef{}
			for _, field := range fields {
				arr = append(arr, field)
			}
			sort.Sort(arr)
			return arr
		},
		"sortedMethods": func(methods map[string]*ServiceMethod) []*ServiceMethod {
			var arr ServiceMethods
			arr = []*ServiceMethod{}
			for _, method := range methods {
				arr = append(arr, method)
			}
			sort.Sort(arr)
			return arr
		},
	}

	protow := new(bytes.Buffer)
	prototpl := template.Must(template.New("main").Funcs(funcMap).Parse(protoTemplate))
	prototpl.Execute(protow, registry)
	fmt.Printf("%s\n-----\n", protow.String())

	/*
		w := new(bytes.Buffer)
		tpl := template.Must(template.New("main").Funcs(funcMap).Parse(typesTemplate))
		tpl.Execute(w, registry)

		data, err := format.Source(w.Bytes())
		if err != nil {
			return ctx, err
		}

		j, err := json.MarshalIndent(registry, "", "  ")
		if err != nil {
			return ctx, err
		}
		fmt.Printf("\n%s\n", j)
		fmt.Printf("%s\n", data)
	*/

	return ctx, nil
}

const typesTemplate = `package types
{{range $structNameRef, $st := .Structs}}
type {{objName $st.Name $st.ExplicitName $st.Private}} struct {
{{- range $fieldNameRef, $field := $st.Fields}}
	{{objName $field.Name $field.ExplicitName $field.Private}} {{$field.GoType}} {{$field.Tags | serializeTags }}
{{- end}}
}
{{end}}
`

const protoTemplate = `
syntax = "proto3";
package proto;
import "google/protobuf/wrappers.proto";
{{range $serviceName, $svc := .Services}}
service {{$svc.Name}} {
{{- range $methodIndex, $method := $svc.Methods | sortedMethods}}
	rpc {{$method.Name}}({{$method.Request}}) returns ({{$method.Response}}) {}
{{- end}}
}
{{end}}
{{range $structNameRef, $st := .Structs}}
{{- if eq $st.IsRoot false}}message {{$st.Name}} {
{{- range $fieldIndex, $field := $st.Fields | sortedFields}}
	{{$field.Prototype}} {{$field.Name}} = {{$fieldIndex | nextNumber}};
{{- end}}
}

{{end -}}
{{- end -}}`
