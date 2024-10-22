package gateway

import (
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"github.com/go-obvious/gateway/internal"
)

func ListenAndServeV2(addr string, h http.Handler) error {
	return internal.ListenAndServe[events.APIGatewayV2HTTPRequest, events.APIGatewayV2HTTPResponse](
		"",
		h,
		internal.ConvertAPIGatewayV2HTTPRequest,
		internal.ConvertResponseV2,
	)
}
