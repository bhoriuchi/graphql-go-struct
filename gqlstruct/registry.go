package gqlstruct

import "strings"

type Registry struct {
	Services         map[string]*Service `json:"services"`
	Structs          map[string]*Def     `json:"structs"`
	RootQuery        string              `json:"root_query"`
	RootMutation     string              `json:"root_mutation"`
	RootSubscription string              `json:"root_subscription"`
}

type Service struct {
	Name    string `json:"name"`
	Methods map[string]*ServiceMethod
}

type ServiceMethod struct {
	Name     string `json:"name"`
	Request  string `json:"request"`
	Response string `json:"response"`
}

// ServiceMethods list of methods
type ServiceMethods []*ServiceMethod

func (f ServiceMethods) Len() int {
	return len(f)
}

func (f ServiceMethods) Less(i, j int) bool {
	return f[i].Name < f[j].Name
}

func (f ServiceMethods) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

type Def struct {
	Name         string               `json:"name"`
	Fields       map[string]*FieldDef `json:"fields"`
	Private      bool                 `json:"private"`
	ExplicitName string               `json:"explicit_name"`
	IsRoot       bool                 `json:"is_root"`
}

type FieldDef struct {
	Ref          string   `json:"ref"`
	Name         string   `json:"name"`
	GoType       string   `json:"gotype"`
	GqlType      string   `json:"gqltype"`
	Prototype    string   `json:"prototype"`
	ProtoIndex   int      `json:"proto_index"`
	Tags         []string `json:"tags"`
	Private      bool     `json:"private"`
	ExplicitName string   `json:"explicit_name"`
}

// FieldDefs list of fields
type FieldDefs []*FieldDef

func (f FieldDefs) Len() int {
	return len(f)
}

func (f FieldDefs) Less(i, j int) bool {
	return f[i].Ref < f[j].Ref
}

func (f FieldDefs) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

type FieldType struct {
	isList    bool
	isNonNull bool
	gqltype   string
	gotype    string
	prototype string
}

func (c *FieldType) String() string {
	if c.gotype == "" {
		return ""
	}

	parts := []string{}
	if c.isList {
		parts = append(parts, "[]")
	}
	if !c.isNonNull {
		parts = append(parts, "*")
	}

	parts = append(parts, c.gotype)
	return strings.Join(parts, "")
}
