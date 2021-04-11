package gqlstruct

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/format"
	"strings"
	"text/template"

	tools "github.com/bhoriuchi/graphql-go-tools"
	"github.com/graphql-go/graphql"
	"github.com/iancoleman/strcase"
)

// Struct directive
const (
	structDirectiveName    = "struct"
	StructDirectiveTypeDef = `directive @struct (
	private: Boolean = false
	omit: Boolean = false
  name: String
	type: String
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
		"type": &graphql.ArgumentConfig{
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
	Private bool     `json:"private"`
	Omit    bool     `json:"omit"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Tags    []string `json:"tags"`
}

var StructDirectiveVisitor = &tools.SchemaDirectiveVisitor{
	VisitObject: func(p tools.VisitObjectParams) {
		reg := p.Context.Value(RegistryKey)
		registry := reg.(*Registry)
		args := &Args{}

		if err := mapProto(p.Args, args); err != nil {
			fmt.Printf("Failed to map arguments: %v\n", err)
			return
		}

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

		for _, field := range p.Node.Fields {
			args, err := getDirectiveConfig(field.Directives)
			if err != nil {
				fmt.Printf("Failed to get field directive args: %v\n", err)
				return
			}

			if args.Omit {
				continue
			}

			fieldName := field.Name.Value
			ft := getFieldType(registry, field.Type, nil)
			obj.Fields[fieldName] = &FieldDef{
				Name:         fieldName,
				GqlType:      ft.gqltype,
				GoType:       ft.String(),
				Private:      args.Private,
				ExplicitName: args.Name,
				Tags:         args.Tags,
			}
		}
	},
}

// Make makes the types
func Make(config tools.ExecutableSchema) error {
	registry := &Registry{
		Structs: map[string]*Def{},
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

	_, err := tools.MakeExecutableSchemaWithContext(ctx, config)
	if err != nil {
		return err
	}

	for _, s := range registry.Structs {
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
	}

	w := new(bytes.Buffer)
	tpl := template.Must(template.New("main").Funcs(funcMap).Parse(typesTemplate))
	tpl.Execute(w, registry)

	data, err := format.Source(w.Bytes())
	if err != nil {
		return err
	}

	j, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("\n%s\n", j)
	fmt.Printf("%s\n", data)

	return nil
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
