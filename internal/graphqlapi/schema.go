package graphqlapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/chrisuzor/commerce-ops-console/internal/service"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

type API struct {
	orders *service.Orders
	schema graphql.Schema
}

func New(orders *service.Orders) (*API, error) {
	api := &API{orders: orders}
	schema, err := api.buildSchema()
	if err != nil {
		return nil, err
	}
	api.schema = schema
	return api, nil
}

func (a *API) Execute(query string, variables map[string]any) *graphql.Result {
	return graphql.Do(graphql.Params{Schema: a.schema, RequestString: query, VariableValues: variables})
}

func (a *API) buildSchema() (graphql.Schema, error) {
	metadataType := graphql.NewScalar(graphql.ScalarConfig{
		Name:         "JSON",
		Serialize:    func(value any) any { return value },
		ParseValue:   func(value any) any { return value },
		ParseLiteral: func(_ ast.Value) any { return nil },
	})

	eventType := graphql.NewObject(graphql.ObjectConfig{Name: "OrderEvent", Fields: graphql.Fields{
		"id":             &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"type":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"source":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"idempotencyKey": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"occurredAt":     &graphql.Field{Type: graphql.NewNonNull(graphql.String), Resolve: timeResolver("OccurredAt")},
		"receivedAt":     &graphql.Field{Type: graphql.NewNonNull(graphql.String), Resolve: timeResolver("ReceivedAt")},
		"metadata":       &graphql.Field{Type: metadataType},
	}})

	issueType := graphql.NewObject(graphql.ObjectConfig{Name: "Issue", Fields: graphql.Fields{
		"code":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"title":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"explanation":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"severity":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"evidence":     &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String)))},
		"acknowledged": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
	}})

	orderType := graphql.NewObject(graphql.ObjectConfig{Name: "Order", Fields: graphql.Fields{
		"id":      &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"status":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"summary": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"events":  &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(eventType)))},
		"issues":  &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(issueType)))},
	}})

	ingestResultType := graphql.NewObject(graphql.ObjectConfig{Name: "IngestResult", Fields: graphql.Fields{
		"duplicate": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"event":     &graphql.Field{Type: graphql.NewNonNull(eventType)},
	}})
	actionType := graphql.NewObject(graphql.ObjectConfig{Name: "Action", Fields: graphql.Fields{
		"id":             &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"orderId":        &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"type":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"actor":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"idempotencyKey": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"createdAt":      &graphql.Field{Type: graphql.NewNonNull(graphql.String), Resolve: timeResolver("CreatedAt")},
	}})
	actionResultType := graphql.NewObject(graphql.ObjectConfig{Name: "ActionResult", Fields: graphql.Fields{
		"action":   &graphql.Field{Type: graphql.NewNonNull(actionType)},
		"replayed": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
	}})

	query := graphql.NewObject(graphql.ObjectConfig{Name: "Query", Fields: graphql.Fields{
		"orders": &graphql.Field{
			Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(orderType))),
			Args: graphql.FieldConfigArgument{"status": &graphql.ArgumentConfig{Type: graphql.String}},
			Resolve: func(params graphql.ResolveParams) (any, error) {
				status, _ := params.Args["status"].(string)
				return a.orders.List(params.Context, status)
			},
		},
		"order": &graphql.Field{
			Type: orderType,
			Args: graphql.FieldConfigArgument{"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)}},
			Resolve: func(params graphql.ResolveParams) (any, error) {
				return a.orders.Get(params.Context, params.Args["id"].(string))
			},
		},
	}})

	eventInput := graphql.NewInputObject(graphql.InputObjectConfig{Name: "EventInput", Fields: graphql.InputObjectConfigFieldMap{
		"orderId":        &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.ID)},
		"type":           &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"source":         &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"idempotencyKey": &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"occurredAt":     &graphql.InputObjectFieldConfig{Type: graphql.NewNonNull(graphql.String)},
		"metadata":       &graphql.InputObjectFieldConfig{Type: metadataType},
	}})

	mutation := graphql.NewObject(graphql.ObjectConfig{Name: "Mutation", Fields: graphql.Fields{
		"ingestEvent": &graphql.Field{
			Type: graphql.NewNonNull(ingestResultType), Args: graphql.FieldConfigArgument{"input": &graphql.ArgumentConfig{Type: graphql.NewNonNull(eventInput)}},
			Resolve: func(params graphql.ResolveParams) (any, error) {
				input, err := parseEventInput(params.Args["input"].(map[string]any))
				if err != nil {
					return nil, err
				}
				return a.orders.Ingest(params.Context, input)
			},
		},
		"retryFulfillment": &graphql.Field{
			Type: graphql.NewNonNull(actionResultType),
			Args: graphql.FieldConfigArgument{
				"orderId":        &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				"actor":          &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				"idempotencyKey": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			},
			Resolve: func(params graphql.ResolveParams) (any, error) {
				return a.orders.RetryFulfillment(params.Context, params.Args["orderId"].(string), params.Args["actor"].(string), params.Args["idempotencyKey"].(string))
			},
		},
		"acknowledgeIssue": &graphql.Field{
			Type: graphql.NewNonNull(graphql.Boolean),
			Args: graphql.FieldConfigArgument{
				"orderId":   &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				"issueCode": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				"actor":     &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			},
			Resolve: func(params graphql.ResolveParams) (any, error) {
				err := a.orders.AcknowledgeIssue(params.Context, params.Args["orderId"].(string), params.Args["issueCode"].(string), params.Args["actor"].(string))
				return err == nil, err
			},
		},
	}})
	return graphql.NewSchema(graphql.SchemaConfig{Query: query, Mutation: mutation})
}

func parseEventInput(raw map[string]any) (service.IngestInput, error) {
	occurredAt, err := time.Parse(time.RFC3339, raw["occurredAt"].(string))
	if err != nil {
		return service.IngestInput{}, errors.New("occurredAt must be RFC3339")
	}
	metadata, _ := raw["metadata"].(map[string]any)
	return service.IngestInput{
		OrderID: raw["orderId"].(string), Type: raw["type"].(string), Source: raw["source"].(string),
		IdempotencyKey: raw["idempotencyKey"].(string), OccurredAt: occurredAt, Metadata: metadata,
	}, nil
}

func timeResolver(field string) graphql.FieldResolveFn {
	return func(params graphql.ResolveParams) (any, error) {
		data, err := json.Marshal(params.Source)
		if err != nil {
			return nil, err
		}
		var values map[string]any
		if err = json.Unmarshal(data, &values); err != nil {
			return nil, err
		}
		value, ok := values[field]
		if !ok {
			value = values[lowerFirst(field)]
		}
		if text, ok := value.(string); ok {
			return text, nil
		}
		return nil, fmt.Errorf("%s is not a timestamp", field)
	}
}

func lowerFirst(value string) string {
	if value == "" {
		return value
	}
	return string(value[0]+('a'-'A')) + value[1:]
}
