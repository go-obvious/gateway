package gateway

import (
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"github.com/go-obvious/gateway/internal"
)

func ListenAndServeV1(addr string, h http.Handler) error {
	return internal.ListenAndServe[events.APIGatewayProxyRequest, events.APIGatewayProxyResponse](
		"",
		h,
		internal.ConvertAPIGatewayProxyRequest,
		internal.ConvertResponseV1,
	)
}
