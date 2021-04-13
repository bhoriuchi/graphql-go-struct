package gqlstruct

import (
	"testing"

	tools "github.com/bhoriuchi/graphql-go-tools"
)

func TestIntrospect(t *testing.T) {
	const typeDefs = `
	type Foo @struct(private: true) {
		name: String!
		description: [String] @struct(
			private: true
			name: "abcDescription"
		)
	}

	type Bar @struct {
		foo_value: [Foo!]
	}
	
	type Query @struct {
		read_foo(
			id: String!
		): Foo @struct(service: "Foo")
	}

	extend type Query @struct {
		list_foo(
			filter: String
		): Foo @struct(service: "Foo")
	}

	type Mutation @struct {
		create_foo(
			name: String!
			description: String
		): Foo @struct(service: "Foo")

		update_foo(
			name: String
			description: String
		): Foo @struct(service: "Foo")
	}
	`
	if _, err := Make(tools.ExecutableSchema{
		TypeDefs: []string{
			typeDefs,
		},
	}); err != nil {
		t.Error(err)
		return
	}
}
