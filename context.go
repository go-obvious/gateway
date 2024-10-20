package gateway

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
)

// key is the type used for any items added to the request context.
type Key int

// requestContextKey is the key for the api gateway proxy `RequestContext`.
const requestContextKey Key = iota

func GetRequestContextKey() Key {
	return requestContextKey
}

// RequestContext returns the APIGatewayV2HTTPRequestContext value stored in ctx.
func RequestContext(ctx context.Context) (events.APIGatewayProxyRequestContext, bool) {
	c, ok := ctx.Value(requestContextKey).(events.APIGatewayProxyRequestContext)
	return c, ok
}

// newContext returns a new Context with specific api gateway v2 values.
func newContext(ctx context.Context, e events.APIGatewayProxyRequest) context.Context {
	return context.WithValue(ctx, requestContextKey, e.RequestContext)
}
