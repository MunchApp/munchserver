package gqlfields

import "github.com/graphql-go/graphql"

// GQL Fields to RETURN to user
var UserType = graphql.NewObject(graphql.ObjectConfig{
	Name: "userFields",
	Fields: graphql.Fields{
		"name":  &graphql.Field{Type: graphql.String},
		"email": &graphql.Field{Type: graphql.String},
		"id":    &graphql.Field{Type: graphql.String},
	},
})