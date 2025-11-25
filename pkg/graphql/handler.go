package graphql

import (
	"encoding/json"
	"net/http"

	"github.com/graphql-go/graphql"
	"github.com/mnohosten/laura-db/pkg/database"
)

// Handler is an HTTP handler for GraphQL requests
type Handler struct {
	schema graphql.Schema
}

// NewHandler creates a new GraphQL HTTP handler
func NewHandler(db *database.Database) (*Handler, error) {
	schema, err := Schema(db)
	if err != nil {
		return nil, err
	}

	return &Handler{
		schema: schema,
	}, nil
}

// GraphQLRequest represents a GraphQL HTTP request
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

// ServeHTTP handles GraphQL HTTP requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "GraphQL only accepts POST requests", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req GraphQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeGraphQLError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Execute GraphQL query
	result := graphql.Do(graphql.Params{
		Schema:         h.schema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		OperationName:  req.OperationName,
		Context:        r.Context(),
	})

	// Write response
	w.Header().Set("Content-Type", "application/json")
	if len(result.Errors) > 0 {
		w.WriteHeader(http.StatusOK) // GraphQL errors still return 200
	}
	json.NewEncoder(w).Encode(result)
}

// writeGraphQLError writes a GraphQL error response
func writeGraphQLError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"errors": []map[string]interface{}{
			{
				"message": message,
			},
		},
	})
}

// GraphiQLHandler returns an HTTP handler for GraphiQL playground
func GraphiQLHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(graphiqlHTML))
	}
}

// graphiqlHTML is the HTML for the GraphiQL playground
const graphiqlHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>LauraDB GraphiQL</title>
    <style>
        body {
            height: 100vh;
            margin: 0;
            width: 100%;
            overflow: hidden;
        }
        #graphiql {
            height: 100vh;
        }
    </style>
    <script crossorigin src="https://unpkg.com/react@17/umd/react.production.min.js"></script>
    <script crossorigin src="https://unpkg.com/react-dom@17/umd/react-dom.production.min.js"></script>
    <link rel="stylesheet" href="https://unpkg.com/graphiql@1.8.7/graphiql.min.css" />
</head>
<body>
    <div id="graphiql">Loading...</div>
    <script src="https://unpkg.com/graphiql@1.8.7/graphiql.min.js" type="application/javascript"></script>
    <script>
        const fetcher = GraphiQL.createFetcher({
            url: '/graphql',
        });

        ReactDOM.render(
            React.createElement(GraphiQL, {
                fetcher: fetcher,
                defaultQuery: '# Welcome to LauraDB GraphQL API\n# \n# Type your queries here. For example:\n# \n# query {\n#   listCollections\n# }\n#\n# mutation {\n#   createCollection(name: "users")\n# }\n',
            }),
            document.getElementById('graphiql'),
        );
    </script>
</body>
</html>
`
