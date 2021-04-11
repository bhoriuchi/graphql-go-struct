package gqlstruct

import (
	"testing"

	tools "github.com/bhoriuchi/graphql-go-tools"
)

func TestIntrospect(t *testing.T) {
	const typeDefs = `
	type Foo @struct(private: true) {
		name: String!
		description: String @struct(
			private: true
			name: "abcDescription"
		)
	}

	type Bar @struct {
		foo_value: Foo!
	}
	
	type Query {
		read_foo(
			id: String!
		): Foo
		create_foo(
			name: String!
			description: String
		): Foo

		update_foo(
			name: String
			description: String
		): Foo
	}
	`
	if err := Make(tools.ExecutableSchema{
		TypeDefs: []string{
			typeDefs,
		},
	}); err != nil {
		t.Error(err)
		return
	}
}
