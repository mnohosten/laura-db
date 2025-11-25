package graphql

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/mnohosten/laura-db/pkg/database"
)

// Schema creates and returns the GraphQL schema for LauraDB
func Schema(db *database.Database) (graphql.Schema, error) {
	// Define the Document type
	documentType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "Document",
		Description: "A document in LauraDB with key-value fields",
		Fields: graphql.Fields{
			"_id": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Unique document identifier",
			},
			"data": &graphql.Field{
				Type:        graphql.NewNonNull(JSONScalar),
				Description: "Document data as JSON",
			},
		},
	})

	// Define the InsertResult type
	insertResultType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "InsertResult",
		Description: "Result of an insert operation",
		Fields: graphql.Fields{
			"insertedId": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "ID of the inserted document",
			},
		},
	})

	// Define the InsertManyResult type
	insertManyResultType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "InsertManyResult",
		Description: "Result of an insertMany operation",
		Fields: graphql.Fields{
			"insertedIds": &graphql.Field{
				Type:        graphql.NewList(graphql.NewNonNull(graphql.String)),
				Description: "IDs of the inserted documents",
			},
			"insertedCount": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "Number of documents inserted",
			},
		},
	})

	// Define the UpdateResult type
	updateResultType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "UpdateResult",
		Description: "Result of an update operation",
		Fields: graphql.Fields{
			"matchedCount": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "Number of documents matched",
			},
			"modifiedCount": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "Number of documents modified",
			},
		},
	})

	// Define the DeleteResult type
	deleteResultType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "DeleteResult",
		Description: "Result of a delete operation",
		Fields: graphql.Fields{
			"deletedCount": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "Number of documents deleted",
			},
		},
	})

	// Define the IndexInfo type
	indexInfoType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "IndexInfo",
		Description: "Information about an index",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Index name",
			},
			"field": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Indexed field",
			},
			"unique": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Boolean),
				Description: "Whether the index is unique",
			},
			"type": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Index type (btree, text, geo, etc.)",
			},
		},
	})

	// Define the CollectionStats type
	collectionStatsType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "CollectionStats",
		Description: "Statistics about a collection",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.String),
				Description: "Collection name",
			},
			"documentCount": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "Total number of documents",
			},
			"indexCount": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "Number of indexes",
			},
		},
	})

	// Define the AggregationResult type
	aggregationResultType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "AggregationResult",
		Description: "Result of an aggregation operation",
		Fields: graphql.Fields{
			"results": &graphql.Field{
				Type:        graphql.NewList(JSONScalar),
				Description: "Aggregation results as JSON array",
			},
		},
	})

	// Create resolver instance
	resolver := NewResolver(db)

	// Define the Query type
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "Query",
		Description: "Root query type for LauraDB",
		Fields: graphql.Fields{
			"findOne": &graphql.Field{
				Type:        documentType,
				Description: "Find a single document by filter",
				Args: graphql.FieldConfigArgument{
					"collection": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
					"filter": &graphql.ArgumentConfig{
						Type:        JSONScalar,
						Description: "Query filter as JSON",
					},
				},
				Resolve: resolver.FindOne,
			},
			"find": &graphql.Field{
				Type:        graphql.NewList(documentType),
				Description: "Find documents matching a filter",
				Args: graphql.FieldConfigArgument{
					"collection": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
					"filter": &graphql.ArgumentConfig{
						Type:        JSONScalar,
						Description: "Query filter as JSON",
					},
					"sort": &graphql.ArgumentConfig{
						Type:        JSONScalar,
						Description: "Sort specification as JSON",
					},
					"limit": &graphql.ArgumentConfig{
						Type:        graphql.Int,
						Description: "Maximum number of documents to return",
					},
					"skip": &graphql.ArgumentConfig{
						Type:        graphql.Int,
						Description: "Number of documents to skip",
					},
				},
				Resolve: resolver.Find,
			},
			"count": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "Count documents matching a filter",
				Args: graphql.FieldConfigArgument{
					"collection": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
					"filter": &graphql.ArgumentConfig{
						Type:        JSONScalar,
						Description: "Query filter as JSON",
					},
				},
				Resolve: resolver.Count,
			},
			"listCollections": &graphql.Field{
				Type:        graphql.NewList(graphql.NewNonNull(graphql.String)),
				Description: "List all collections in the database",
				Resolve:     resolver.ListCollections,
			},
			"collectionStats": &graphql.Field{
				Type:        collectionStatsType,
				Description: "Get statistics for a collection",
				Args: graphql.FieldConfigArgument{
					"collection": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
				},
				Resolve: resolver.CollectionStats,
			},
			"listIndexes": &graphql.Field{
				Type:        graphql.NewList(indexInfoType),
				Description: "List all indexes in a collection",
				Args: graphql.FieldConfigArgument{
					"collection": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
				},
				Resolve: resolver.ListIndexes,
			},
			"aggregate": &graphql.Field{
				Type:        aggregationResultType,
				Description: "Run an aggregation pipeline",
				Args: graphql.FieldConfigArgument{
					"collection": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
					"pipeline": &graphql.ArgumentConfig{
						Type:        graphql.NewList(JSONScalar),
						Description: "Aggregation pipeline stages",
					},
				},
				Resolve: resolver.Aggregate,
			},
		},
	})

	// Define the Mutation type
	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "Mutation",
		Description: "Root mutation type for LauraDB",
		Fields: graphql.Fields{
			"createCollection": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Boolean),
				Description: "Create a new collection",
				Args: graphql.FieldConfigArgument{
					"name": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
				},
				Resolve: resolver.CreateCollection,
			},
			"dropCollection": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Boolean),
				Description: "Drop a collection",
				Args: graphql.FieldConfigArgument{
					"name": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
				},
				Resolve: resolver.DropCollection,
			},
			"insertOne": &graphql.Field{
				Type:        insertResultType,
				Description: "Insert a single document",
				Args: graphql.FieldConfigArgument{
					"collection": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
					"document": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(JSONScalar),
						Description: "Document to insert",
					},
				},
				Resolve: resolver.InsertOne,
			},
			"insertMany": &graphql.Field{
				Type:        insertManyResultType,
				Description: "Insert multiple documents",
				Args: graphql.FieldConfigArgument{
					"collection": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
					"documents": &graphql.ArgumentConfig{
						Type:        graphql.NewList(graphql.NewNonNull(JSONScalar)),
						Description: "Documents to insert",
					},
				},
				Resolve: resolver.InsertMany,
			},
			"updateOne": &graphql.Field{
				Type:        updateResultType,
				Description: "Update a single document",
				Args: graphql.FieldConfigArgument{
					"collection": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
					"filter": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(JSONScalar),
						Description: "Query filter",
					},
					"update": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(JSONScalar),
						Description: "Update operations",
					},
				},
				Resolve: resolver.UpdateOne,
			},
			"updateMany": &graphql.Field{
				Type:        updateResultType,
				Description: "Update multiple documents",
				Args: graphql.FieldConfigArgument{
					"collection": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
					"filter": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(JSONScalar),
						Description: "Query filter",
					},
					"update": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(JSONScalar),
						Description: "Update operations",
					},
				},
				Resolve: resolver.UpdateMany,
			},
			"deleteOne": &graphql.Field{
				Type:        deleteResultType,
				Description: "Delete a single document",
				Args: graphql.FieldConfigArgument{
					"collection": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
					"filter": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(JSONScalar),
						Description: "Query filter",
					},
				},
				Resolve: resolver.DeleteOne,
			},
			"deleteMany": &graphql.Field{
				Type:        deleteResultType,
				Description: "Delete multiple documents",
				Args: graphql.FieldConfigArgument{
					"collection": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
					"filter": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(JSONScalar),
						Description: "Query filter",
					},
				},
				Resolve: resolver.DeleteMany,
			},
			"createIndex": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Boolean),
				Description: "Create an index",
				Args: graphql.FieldConfigArgument{
					"collection": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
					"field": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Field to index",
					},
					"unique": &graphql.ArgumentConfig{
						Type:         graphql.Boolean,
						Description:  "Whether the index should be unique",
						DefaultValue: false,
					},
					"name": &graphql.ArgumentConfig{
						Type:        graphql.String,
						Description: "Index name (optional)",
					},
				},
				Resolve: resolver.CreateIndex,
			},
			"dropIndex": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Boolean),
				Description: "Drop an index",
				Args: graphql.FieldConfigArgument{
					"collection": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name",
					},
					"name": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Index name",
					},
				},
				Resolve: resolver.DropIndex,
			},
		},
	})

	// Define the Subscription type
	subscriptionType := graphql.NewObject(graphql.ObjectConfig{
		Name:        "Subscription",
		Description: "Root subscription type for LauraDB",
		Fields: graphql.Fields{
			"watchCollection": &graphql.Field{
				Type:        documentType,
				Description: "Watch for changes in a collection",
				Args: graphql.FieldConfigArgument{
					"collection": &graphql.ArgumentConfig{
						Type:        graphql.NewNonNull(graphql.String),
						Description: "Collection name to watch",
					},
					"filter": &graphql.ArgumentConfig{
						Type:        JSONScalar,
						Description: "Optional filter for changes",
					},
				},
				Resolve: resolver.WatchCollection,
			},
		},
	})

	// Create the schema
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:        queryType,
		Mutation:     mutationType,
		Subscription: subscriptionType,
	})

	if err != nil {
		return graphql.Schema{}, fmt.Errorf("failed to create GraphQL schema: %w", err)
	}

	return schema, nil
}
