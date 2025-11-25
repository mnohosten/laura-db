package graphql

import (
	"encoding/json"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

// JSONScalar is a custom GraphQL scalar type for JSON data
var JSONScalar = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "JSON",
	Description: "The `JSON` scalar type represents JSON values as specified by ECMA-404",
	// Serialize converts Go value to JSON
	Serialize: func(value interface{}) interface{} {
		return value
	},
	// ParseValue converts incoming variables to Go value
	ParseValue: func(value interface{}) interface{} {
		// Handle nil
		if value == nil {
			return nil
		}

		// If it's already a map or slice, return as is
		switch v := value.(type) {
		case map[string]interface{}:
			return v
		case []interface{}:
			return v
		case string:
			// Try to parse as JSON
			var result interface{}
			if err := json.Unmarshal([]byte(v), &result); err != nil {
				return nil
			}
			return result
		default:
			// For other types, return as is
			return value
		}
	},
	// ParseLiteral converts AST literal to Go value
	ParseLiteral: func(valueAST ast.Value) interface{} {
		switch valueAST := valueAST.(type) {
		case *ast.ObjectValue:
			obj := make(map[string]interface{})
			for _, field := range valueAST.Fields {
				obj[field.Name.Value] = parseLiteralValue(field.Value)
			}
			return obj
		case *ast.ListValue:
			list := make([]interface{}, len(valueAST.Values))
			for i, value := range valueAST.Values {
				list[i] = parseLiteralValue(value)
			}
			return list
		case *ast.StringValue:
			return valueAST.Value
		case *ast.IntValue:
			// Try to convert to int64
			var num int64
			fmt.Sscanf(valueAST.Value, "%d", &num)
			return num
		case *ast.FloatValue:
			// Try to convert to float64
			var num float64
			fmt.Sscanf(valueAST.Value, "%f", &num)
			return num
		case *ast.BooleanValue:
			return valueAST.Value
		case *ast.EnumValue:
			return valueAST.Value
		default:
			return nil
		}
	},
})

// parseLiteralValue is a helper to recursively parse AST values
func parseLiteralValue(valueAST ast.Value) interface{} {
	switch valueAST := valueAST.(type) {
	case *ast.ObjectValue:
		obj := make(map[string]interface{})
		for _, field := range valueAST.Fields {
			obj[field.Name.Value] = parseLiteralValue(field.Value)
		}
		return obj
	case *ast.ListValue:
		list := make([]interface{}, len(valueAST.Values))
		for i, value := range valueAST.Values {
			list[i] = parseLiteralValue(value)
		}
		return list
	case *ast.StringValue:
		return valueAST.Value
	case *ast.IntValue:
		var num int64
		fmt.Sscanf(valueAST.Value, "%d", &num)
		return num
	case *ast.FloatValue:
		var num float64
		fmt.Sscanf(valueAST.Value, "%f", &num)
		return num
	case *ast.BooleanValue:
		return valueAST.Value
	case *ast.EnumValue:
		return valueAST.Value
	default:
		return nil
	}
}
