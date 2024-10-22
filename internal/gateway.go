package internal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/pkg/errors"
)

// ===========================
// Context Handling
// ===========================

// Key is the type used for any items added to the request context.
type Key int

// requestContextKey is the key for the API Gateway proxy `RequestContext`.
const requestContextKey Key = iota

// GetRequestContextKey returns the key used for storing the RequestContext in the context.
func GetRequestContextKey() Key {
	return requestContextKey
}

// RequestContext retrieves the RequestContext from the context.
func RequestContext[T any](ctx context.Context) (T, bool) {
	c, ok := ctx.Value(requestContextKey).(T)
	return c, ok
}

// NewContext returns a new Context with the API Gateway proxy RequestContext.
func NewContext[T any](ctx context.Context, e T) context.Context {
	return context.WithValue(ctx, requestContextKey, e)
}

// ===========================
// Converter Function Types
// ===========================

// RequestConverter is a function type that converts an event of type T to an *http.Request
type RequestConverter[T any] func(context.Context, T) (*http.Request, error)

// ResponseConverter is a function type that converts ResponseData to a response of type R
type ResponseConverter[R any] func(ResponseData) (R, error)

// ===========================
// Response Data Struct
// ===========================

// ResponseData captures the HTTP response data
type ResponseData struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// ===========================
// Gateway Struct and Methods
// ===========================

// Gateway is a generic struct that wraps an http.Handler and converter functions
type Gateway[T any, R any] struct {
	handler           http.Handler
	requestConverter  RequestConverter[T]
	responseConverter ResponseConverter[R]
}

// NewGateway creates a new Gateway with the given handler and converters
func NewGateway[T any, R any](handler http.Handler, requestConverter RequestConverter[T], responseConverter ResponseConverter[R]) *Gateway[T, R] {
	return &Gateway[T, R]{handler: handler, requestConverter: requestConverter, responseConverter: responseConverter}
}

// Invoke handles the Lambda invocation by converting the event to an HTTP request,
// processing it, and converting the response back to the Lambda response format.
func (gw *Gateway[T, R]) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	var evt T

	// Unmarshal the payload into the generic event type T
	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Convert the event to an *http.Request using the converter function
	req, err := gw.requestConverter(ctx, evt)
	if err != nil {
		return nil, fmt.Errorf("failed to convert event to request: %w", err)
	}

	// Create a ResponseWriter to capture the response
	w := NewResponse()

	// Serve the HTTP request using the provided handler
	gw.handler.ServeHTTP(w, req)

	// Prepare the response data
	respData := ResponseData{
		StatusCode: w.statusCode,
		Headers:    w.Header(),
		Body:       w.buf.Bytes(),
	}

	// Convert the response data to the desired response type R
	resp, err := gw.responseConverter(respData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	// Marshal the response back to JSON
	return json.Marshal(resp)
}

// ===========================
// ListenAndServe Function
// ===========================

// ListenAndServe is a generic function that sets up the Gateway and starts the Lambda handler
func ListenAndServe[T any, R any](addr string, handler http.Handler, requestConverter RequestConverter[T], responseConverter ResponseConverter[R]) error {
	if handler == nil {
		handler = http.DefaultServeMux
	}

	gw := NewGateway[T, R](handler, requestConverter, responseConverter)

	lambda.StartHandler(gw)

	return nil
}

// ===========================
// Custom ResponseWriter
// ===========================

// ResponseWriter implements the http.ResponseWriter interface to capture HTTP responses
type ResponseWriter struct {
	buf           bytes.Buffer
	header        http.Header
	wroteHeader   bool
	statusCode    int
	closeNotifyCh chan bool
}

// NewResponse creates a new ResponseWriter instance.
func NewResponse() *ResponseWriter {
	return &ResponseWriter{
		header:        make(http.Header),
		statusCode:    http.StatusOK,
		closeNotifyCh: make(chan bool, 1),
	}
}

// Header returns the header map that will be sent by WriteHeader.
func (w *ResponseWriter) Header() http.Header {
	return w.header
}

// Write writes the data to the buffer.
func (w *ResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.buf.Write(b)
}

// WriteHeader sends an HTTP response header with the provided status code.
func (w *ResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	if w.header.Get("Content-Type") == "" {
		w.header.Set("Content-Type", "text/plain; charset=utf8")
	}
	w.statusCode = statusCode
	w.wroteHeader = true
}

// CloseNotify notifies when the response is closed.
func (w *ResponseWriter) CloseNotify() <-chan bool {
	return w.closeNotifyCh
}

// ===========================
// Helper Functions
// ===========================

// isBinary returns true if the response represents binary data.
func isBinary(h http.Header) bool {
	contentType := h.Get("Content-Type")
	return !isTextMime(contentType) || h.Get("Content-Encoding") == "gzip"
}

// isTextMime returns true if the content type represents textual data.
func isTextMime(kind string) bool {
	mt, _, err := mime.ParseMediaType(kind)
	if err != nil {
		return false
	}

	if strings.HasPrefix(mt, "text/") {
		return true
	}

	switch mt {
	case "image/svg+xml", "application/json", "application/xml", "application/javascript", "application/vnd.api+json":
		return true
	default:
		return false
	}
}

// ===========================
// Response Converter Functions
// ===========================

// ConvertResponseV1 converts ResponseData to APIGatewayProxyResponse (v1)
func ConvertResponseV1(data ResponseData) (events.APIGatewayProxyResponse, error) {
	out := events.APIGatewayProxyResponse{
		StatusCode:        data.StatusCode,
		Headers:           make(map[string]string),
		MultiValueHeaders: make(map[string][]string),
	}

	for k, v := range data.Headers {
		if len(v) == 1 {
			out.Headers[k] = v[0]
			out.MultiValueHeaders[k] = v
		} else if len(v) > 1 {
			out.MultiValueHeaders[k] = v
		}
	}

	isBin := isBinary(data.Headers)

	out.IsBase64Encoded = isBin

	if isBin {
		out.Body = base64.StdEncoding.EncodeToString(data.Body)
	} else {
		out.Body = string(data.Body)
	}

	return out, nil
}

// ConvertResponseV2 converts ResponseData to APIGatewayV2HTTPResponse (v2)
func ConvertResponseV2(data ResponseData) (events.APIGatewayV2HTTPResponse, error) {
	out := events.APIGatewayV2HTTPResponse{
		StatusCode:        data.StatusCode,
		Headers:           make(map[string]string),
		MultiValueHeaders: make(map[string][]string),
		Cookies:           []string{},
	}

	for k, v := range data.Headers {
		if http.CanonicalHeaderKey(k) == "Set-Cookie" {
			out.Cookies = append(out.Cookies, v...)
		} else if len(v) == 1 {
			out.Headers[k] = v[0]
			out.MultiValueHeaders[k] = v
		} else if len(v) > 1 {
			out.MultiValueHeaders[k] = v
		}
	}

	isBin := isBinary(data.Headers)

	out.IsBase64Encoded = isBin

	if isBin {
		out.Body = base64.StdEncoding.EncodeToString(data.Body)
	} else {
		out.Body = string(data.Body)
	}

	return out, nil
}

// ===========================
// Request Converter Functions
// ===========================

// ConvertAPIGatewayProxyRequest converts APIGatewayProxyRequest (v1) to *http.Request
func ConvertAPIGatewayProxyRequest(ctx context.Context, e events.APIGatewayProxyRequest) (*http.Request, error) {
	// Parse the path
	u, err := url.Parse(e.Path)
	if err != nil {
		return nil, errors.Wrap(err, "parsing path")
	}

	// Build query parameters
	q := u.Query()
	for k, v := range e.QueryStringParameters {
		q.Set(k, v)
	}
	for k, values := range e.MultiValueQueryStringParameters {
		q[k] = values
	}
	u.RawQuery = q.Encode()

	// Decode the body if it's base64 encoded
	body := e.Body
	if e.IsBase64Encoded {
		b, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			return nil, errors.Wrap(err, "decoding base64 body")
		}
		body = string(b)
	}

	// Create a new HTTP request
	req, err := http.NewRequest(e.HTTPMethod, u.String(), strings.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "creating request")
	}

	// Manually set RequestURI
	req.RequestURI = u.RequestURI()

	// Set RemoteAddr
	req.RemoteAddr = e.RequestContext.Identity.SourceIP

	// Set headers
	for k, v := range e.Headers {
		req.Header.Set(k, v)
	}
	for k, values := range e.MultiValueHeaders {
		for _, v := range values {
			req.Header.Add(k, v)
		}
	}

	// Set Content-Length if not already set
	if req.Header.Get("Content-Length") == "" && body != "" {
		req.Header.Set("Content-Length", strconv.Itoa(len(body)))
	}

	// Set custom headers
	req.Header.Set("X-Request-Id", e.RequestContext.RequestID)
	req.Header.Set("X-Stage", e.RequestContext.Stage)

	// Add custom context values
	req = req.WithContext(NewContext(ctx, e))

	// X-Ray support
	if traceID := ctx.Value("x-amzn-trace-id"); traceID != nil {
		req.Header.Set("X-Amzn-Trace-Id", fmt.Sprintf("%v", traceID))
	}

	// Set Host
	req.URL.Host = req.Header.Get("Host")
	req.Host = req.URL.Host

	return req, nil
}

// ConvertAPIGatewayV2HTTPRequest converts APIGatewayV2HTTPRequest (v2) to *http.Request
func ConvertAPIGatewayV2HTTPRequest(ctx context.Context, e events.APIGatewayV2HTTPRequest) (*http.Request, error) {
	// Parse the raw path
	u, err := url.Parse(e.RawPath)
	if err != nil {
		return nil, errors.Wrap(err, "parsing raw path")
	}

	// Set the raw query string
	u.RawQuery = e.RawQueryString

	// Decode the body if it's base64 encoded
	body := e.Body
	if e.IsBase64Encoded {
		b, err := base64.StdEncoding.DecodeString(body)
		if err != nil {
			return nil, errors.Wrap(err, "decoding base64 body")
		}
		body = string(b)
	}

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, e.RequestContext.HTTP.Method, u.String(), strings.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "creating request")
	}

	// Manually set RequestURI
	req.RequestURI = u.RequestURI()

	// Set RemoteAddr
	req.RemoteAddr = e.RequestContext.HTTP.SourceIP

	// Set headers
	for k, values := range e.Headers {
		for _, v := range strings.Split(values, ",") {
			req.Header.Add(k, strings.TrimSpace(v))
		}
	}
	for _, c := range e.Cookies {
		req.Header.Add("Cookie", c)
	}

	// Set Content-Length if not already set
	if req.Header.Get("Content-Length") == "" && body != "" {
		req.Header.Set("Content-Length", strconv.Itoa(len(body)))
	}

	// Set custom headers
	req.Header.Set("X-Request-Id", e.RequestContext.RequestID)
	req.Header.Set("X-Stage", e.RequestContext.Stage)

	// Add custom context values
	req = req.WithContext(NewContext(ctx, e))

	// X-Ray support
	if traceID := ctx.Value("x-amzn-trace-id"); traceID != nil {
		req.Header.Set("X-Amzn-Trace-Id", fmt.Sprintf("%v", traceID))
	}

	// Set Host
	req.URL.Host = req.Header.Get("Host")
	req.Host = req.URL.Host

	return req, nil
}
