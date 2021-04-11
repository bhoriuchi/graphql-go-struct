package gqlstruct

import "strings"

type Registry struct {
	Structs map[string]*Def `json:"structs"`
}

type Def struct {
	Name         string               `json:"name"`
	Fields       map[string]*FieldDef `json:"fields"`
	Private      bool                 `json:"private"`
	ExplicitName string               `json:"explicit_name"`
}

type FieldDef struct {
	Name         string   `json:"name"`
	GoType       string   `json:"gotype"`
	GqlType      string   `json:"gqltype"`
	Tags         []string `json:"tags"`
	Private      bool     `json:"private"`
	ExplicitName string   `json:"explicit_name"`
}

type FieldType struct {
	isList   bool
	isNonNul bool
	gqltype  string
	gotype   string
}

func (c *FieldType) String() string {
	if c.gotype == "" {
		return ""
	}

	parts := []string{}
	if c.isList {
		parts = append(parts, "[]")
	}
	if !c.isNonNul {
		parts = append(parts, "*")
	}

	parts = append(parts, c.gotype)
	return strings.Join(parts, "")
}
