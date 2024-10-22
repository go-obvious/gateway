package internal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestNewContext(t *testing.T) {
	type testEvent struct {
		ID string
	}

	ctx := context.Background()
	event := testEvent{ID: "123"}

	newCtx := NewContext(ctx, event)

	retrievedEvent, ok := RequestContext[testEvent](newCtx)
	if !ok {
		t.Fatalf("expected to retrieve event from context")
	}

	if retrievedEvent.ID != event.ID {
		t.Errorf("expected event ID %s, got %s", event.ID, retrievedEvent.ID)
	}
}

func TestNewGateway(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	requestConverter := func(ctx context.Context, e events.APIGatewayProxyRequest) (*http.Request, error) {
		return ConvertAPIGatewayProxyRequest(ctx, e)
	}

	responseConverter := func(data ResponseData) (events.APIGatewayProxyResponse, error) {
		return ConvertResponseV1(data)
	}

	gw := NewGateway(handler, requestConverter, responseConverter)

	if gw == nil {
		t.Fatalf("expected NewGateway to return a non-nil Gateway")
	}

	if gw.handler == nil {
		t.Errorf("expected handler to be set")
	}

	if gw.requestConverter == nil {
		t.Errorf("expected requestConverter to be set")
	}

	if gw.responseConverter == nil {
		t.Errorf("expected responseConverter to be set")
	}
}

func TestGateway_Invoke(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	requestConverter := func(ctx context.Context, e events.APIGatewayProxyRequest) (*http.Request, error) {
		return ConvertAPIGatewayProxyRequest(ctx, e)
	}

	responseConverter := func(data ResponseData) (events.APIGatewayProxyResponse, error) {
		return ConvertResponseV1(data)
	}

	gw := NewGateway(handler, requestConverter, responseConverter)

	event := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/",
	}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	respPayload, err := gw.Invoke(context.Background(), payload)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	var resp events.APIGatewayProxyResponse
	if err := json.Unmarshal(respPayload, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	expectedBody := "Hello, World!"
	if resp.Body != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, resp.Body)
	}
}

func TestGateway_InvokeV2(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	requestConverter := func(ctx context.Context, e events.APIGatewayV2HTTPRequest) (*http.Request, error) {
		return ConvertAPIGatewayV2HTTPRequest(ctx, e)
	}

	responseConverter := func(data ResponseData) (events.APIGatewayV2HTTPResponse, error) {
		return ConvertResponseV2(data)
	}

	gw := NewGateway(handler, requestConverter, responseConverter)

	event := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "GET",
			},
		},
		RawPath:        "/",
		RawQueryString: "",
	}

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	respPayload, err := gw.Invoke(context.Background(), payload)
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	var resp events.APIGatewayV2HTTPResponse
	if err := json.Unmarshal(respPayload, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	expectedBody := "Hello, World!"
	if resp.Body != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, resp.Body)
	}
}

func TestConvertAPIGatewayProxyRequest(t *testing.T) {
	event := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/test",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:            `{"key":"value"}`,
		IsBase64Encoded: false,
	}

	req, err := ConvertAPIGatewayProxyRequest(context.Background(), event)
	if err != nil {
		t.Fatalf("ConvertAPIGatewayProxyRequest failed: %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("expected method GET, got %s", req.Method)
	}

	if req.URL.Path != "/test" {
		t.Errorf("expected path /test, got %s", req.URL.Path)
	}

	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
	}
}

func TestConvertAPIGatewayV2HTTPRequest(t *testing.T) {
	event := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "POST",
			},
		},
		RawPath:        "/test",
		RawQueryString: "param=value",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:            `{"key":"value"}`,
		IsBase64Encoded: false,
	}

	req, err := ConvertAPIGatewayV2HTTPRequest(context.Background(), event)
	if err != nil {
		t.Fatalf("ConvertAPIGatewayV2HTTPRequest failed: %v", err)
	}

	if req.Method != "POST" {
		t.Errorf("expected method POST, got %s", req.Method)
	}

	if req.URL.Path != "/test" {
		t.Errorf("expected path /test, got %s", req.URL.Path)
	}

	if req.URL.RawQuery != "param=value" {
		t.Errorf("expected query param=value, got %s", req.URL.RawQuery)
	}

	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
	}
}

// Additional Tests

func TestIsBinary(t *testing.T) {
	tests := []struct {
		headers  http.Header
		expected bool
	}{
		{http.Header{"Content-Type": []string{"application/json"}}, false},
		{http.Header{"Content-Type": []string{"image/png"}}, true},
		{http.Header{"Content-Encoding": []string{"gzip"}}, true},
	}

	for _, test := range tests {
		result := isBinary(test.headers)
		if result != test.expected {
			t.Errorf("expected %v, got %v", test.expected, result)
		}
	}
}

func TestIsTextMime(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"text/plain", true},
		{"application/json", true},
		{"image/png", false},
	}

	for _, test := range tests {
		result := isTextMime(test.contentType)
		if result != test.expected {
			t.Errorf("expected %v, got %v", test.expected, result)
		}
	}
}

func TestResponseWriter(t *testing.T) {
	w := NewResponse()

	if w.Header() == nil {
		t.Errorf("expected non-nil header")
	}

	w.WriteHeader(http.StatusNotFound)
	if w.statusCode != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d", http.StatusNotFound, w.statusCode)
	}

	body := []byte("test body")
	w.Write(body)
	if w.buf.String() != string(body) {
		t.Errorf("expected body %q, got %q", string(body), w.buf.String())
	}
}

func TestConvertAPIGatewayProxyRequest_MultiHeaders(t *testing.T) {
	event := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/test",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		MultiValueHeaders: map[string][]string{
			"X-Custom-Header": {"value1", "value2"},
		},
		Body:            `{"key":"value"}`,
		IsBase64Encoded: false,
	}

	req, err := ConvertAPIGatewayProxyRequest(context.Background(), event)
	if err != nil {
		t.Fatalf("ConvertAPIGatewayProxyRequest failed: %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("expected method GET, got %s", req.Method)
	}

	if req.URL.Path != "/test" {
		t.Errorf("expected path /test, got %s", req.URL.Path)
	}

	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
	}

	if req.Header["X-Custom-Header"][0] != "value1" || req.Header["X-Custom-Header"][1] != "value2" {
		t.Errorf("expected X-Custom-Header to have values [value1, value2], got %v", req.Header["X-Custom-Header"])
	}
}

func TestConvertResponseV2(t *testing.T) {
	tests := []struct {
		name     string
		data     ResponseData
		expected events.APIGatewayV2HTTPResponse
	}{
		{
			name: "text response",
			data: ResponseData{
				StatusCode: http.StatusOK,
				Headers:    http.Header{"Content-Type": []string{"text/plain"}},
				Body:       []byte("Hello, World!"),
			},
			expected: events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusOK,
				Headers:    map[string]string{"Content-Type": "text/plain"},
				Body:       "Hello, World!",
			},
		},
		{
			name: "json response",
			data: ResponseData{
				StatusCode: http.StatusOK,
				Headers:    http.Header{"Content-Type": []string{"application/json"}},
				Body:       []byte(`{"message":"Hello, World!"}`),
			},
			expected: events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusOK,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       `{"message":"Hello, World!"}`,
			},
		},
		{
			name: "binary response",
			data: ResponseData{
				StatusCode: http.StatusOK,
				Headers:    http.Header{"Content-Type": []string{"image/png"}},
				Body:       []byte{0x89, 0x50, 0x4E, 0x47},
			},
			expected: events.APIGatewayV2HTTPResponse{
				StatusCode:      http.StatusOK,
				Headers:         map[string]string{"Content-Type": "image/png"},
				Body:            base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4E, 0x47}),
				IsBase64Encoded: true,
			},
		},
		{
			name: "response with cookies",
			data: ResponseData{
				StatusCode: http.StatusOK,
				Headers:    http.Header{"Set-Cookie": []string{"cookie1=value1", "cookie2=value2"}},
				Body:       []byte("Hello, World!"),
			},
			expected: events.APIGatewayV2HTTPResponse{
				StatusCode:      http.StatusOK,
				Headers:         map[string]string{},
				Cookies:         []string{"cookie1=value1", "cookie2=value2"},
				Body:            base64.StdEncoding.EncodeToString([]byte("Hello, World!")),
				IsBase64Encoded: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := ConvertResponseV2(tt.data)
			if err != nil {
				t.Fatalf("ConvertResponseV2 failed: %v", err)
			}

			if resp.StatusCode != tt.expected.StatusCode {
				t.Errorf("expected status code %d, got %d", tt.expected.StatusCode, resp.StatusCode)
			}

			if resp.Body != tt.expected.Body {
				t.Errorf("expected body %q, got %q", tt.expected.Body, resp.Body)
			}

			if resp.IsBase64Encoded != tt.expected.IsBase64Encoded {
				t.Errorf("expected IsBase64Encoded %v, got %v", tt.expected.IsBase64Encoded, resp.IsBase64Encoded)
			}

			for k, v := range tt.expected.Headers {
				if resp.Headers[k] != v {
					t.Errorf("expected header %s to be %q, got %q", k, v, resp.Headers[k])
				}
			}

			if len(resp.Cookies) != len(tt.expected.Cookies) {
				t.Errorf("expected %d cookies, got %d", len(tt.expected.Cookies), len(resp.Cookies))
			}

			for i, cookie := range tt.expected.Cookies {
				if resp.Cookies[i] != cookie {
					t.Errorf("expected cookie %q, got %q", cookie, resp.Cookies[i])
				}
			}
		})
	}
}
func TestConvertResponseV2_SingleHeader(t *testing.T) {
	data := ResponseData{
		StatusCode: http.StatusOK,
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(`{"message":"Hello, World!"}`),
	}

	expected := events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       `{"message":"Hello, World!"}`,
	}

	resp, err := ConvertResponseV2(data)
	if err != nil {
		t.Fatalf("ConvertResponseV2 failed: %v", err)
	}

	if resp.StatusCode != expected.StatusCode {
		t.Errorf("expected status code %d, got %d", expected.StatusCode, resp.StatusCode)
	}

	if resp.Body != expected.Body {
		t.Errorf("expected body %q, got %q", expected.Body, resp.Body)
	}

	for k, v := range expected.Headers {
		if resp.Headers[k] != v {
			t.Errorf("expected header %s to be %q, got %q", k, v, resp.Headers[k])
		}
	}
}

func TestConvertAPIGatewayProxyRequest_Base64Body(t *testing.T) {
	encodedBody := base64.StdEncoding.EncodeToString([]byte(`{"key":"value"}`))
	event := events.APIGatewayProxyRequest{
		HTTPMethod: "POST",
		Path:       "/test",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:            encodedBody,
		IsBase64Encoded: true,
	}

	req, err := ConvertAPIGatewayProxyRequest(context.Background(), event)
	if err != nil {
		t.Fatalf("ConvertAPIGatewayProxyRequest failed: %v", err)
	}

	if req.Method != "POST" {
		t.Errorf("expected method POST, got %s", req.Method)
	}

	if req.URL.Path != "/test" {
		t.Errorf("expected path /test, got %s", req.URL.Path)
	}

	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	expectedBody := `{"key":"value"}`
	if string(body) != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, string(body))
	}
}

func TestConvertAPIGatewayV2HTTPRequest_Base64Body(t *testing.T) {
	encodedBody := base64.StdEncoding.EncodeToString([]byte(`{"key":"value"}`))
	event := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "POST",
			},
		},
		RawPath:        "/test",
		RawQueryString: "param=value",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:            encodedBody,
		IsBase64Encoded: true,
	}

	req, err := ConvertAPIGatewayV2HTTPRequest(context.Background(), event)
	if err != nil {
		t.Fatalf("ConvertAPIGatewayV2HTTPRequest failed: %v", err)
	}

	if req.Method != "POST" {
		t.Errorf("expected method POST, got %s", req.Method)
	}

	if req.URL.Path != "/test" {
		t.Errorf("expected path /test, got %s", req.URL.Path)
	}

	if req.URL.RawQuery != "param=value" {
		t.Errorf("expected query param=value, got %s", req.URL.RawQuery)
	}

	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	expectedBody := `{"key":"value"}`
	if string(body) != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, string(body))
	}
}

func TestConvertAPIGatewayV2HTTPRequest_Cookies(t *testing.T) {
	event := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "GET",
			},
		},
		RawPath:        "/test",
		RawQueryString: "param=value",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Cookies:         []string{"cookie1=value1", "cookie2=value2"},
		Body:            `{"key":"value"}`,
		IsBase64Encoded: false,
	}

	req, err := ConvertAPIGatewayV2HTTPRequest(context.Background(), event)
	if err != nil {
		t.Fatalf("ConvertAPIGatewayV2HTTPRequest failed: %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("expected method GET, got %s", req.Method)
	}

	if req.URL.Path != "/test" {
		t.Errorf("expected path /test, got %s", req.URL.Path)
	}

	if req.URL.RawQuery != "param=value" {
		t.Errorf("expected query param=value, got %s", req.URL.RawQuery)
	}

	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
	}

	cookies := req.Header["Cookie"]
	if len(cookies) != 2 || cookies[0] != "cookie1=value1" || cookies[1] != "cookie2=value2" {
		t.Errorf("expected cookies [cookie1=value1, cookie2=value2], got %v", cookies)
	}
}

func TestConvertAPIGatewayProxyRequest_QueryString(t *testing.T) {
	event := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/test",
		QueryStringParameters: map[string]string{
			"param1": "value1",
			"param2": "value2",
		},
		MultiValueQueryStringParameters: map[string][]string{
			"param3": {"value3a", "value3b"},
		},
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:            `{"key":"value"}`,
		IsBase64Encoded: false,
	}

	req, err := ConvertAPIGatewayProxyRequest(context.Background(), event)
	if err != nil {
		t.Fatalf("ConvertAPIGatewayProxyRequest failed: %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("expected method GET, got %s", req.Method)
	}

	if req.URL.Path != "/test" {
		t.Errorf("expected path /test, got %s", req.URL.Path)
	}

	expectedQuery := "param1=value1&param2=value2&param3=value3a&param3=value3b"
	if req.URL.RawQuery != expectedQuery {
		t.Errorf("expected query %s, got %s", expectedQuery, req.URL.RawQuery)
	}

	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
	}
}

func TestConvertAPIGatewayV2HTTPRequest_TraceID(t *testing.T) {
	encodedBody := base64.StdEncoding.EncodeToString([]byte(`{"key":"value"}`))
	event := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "POST",
			},
			RequestID: "test-request-id",
			Stage:     "test-stage",
		},
		RawPath:        "/test",
		RawQueryString: "param=value",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Cookies:         []string{"cookie1=value1", "cookie2=value2"},
		Body:            encodedBody,
		IsBase64Encoded: true,
	}

	const traceIDKey string = "x-amzn-trace-id"
	ctx := context.WithValue(context.Background(), traceIDKey, "test-trace-id")

	req, err := ConvertAPIGatewayV2HTTPRequest(ctx, event)
	if err != nil {
		t.Fatalf("ConvertAPIGatewayV2HTTPRequest failed: %v", err)
	}

	if req.Method != "POST" {
		t.Errorf("expected method POST, got %s", req.Method)
	}

	if req.URL.Path != "/test" {
		t.Errorf("expected path /test, got %s", req.URL.Path)
	}

	if req.URL.RawQuery != "param=value" {
		t.Errorf("expected query param=value, got %s", req.URL.RawQuery)
	}

	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
	}

	if req.Header.Get("X-Amzn-Trace-Id") != "test-trace-id" {
		t.Errorf("expected X-Amzn-Trace-Id test-trace-id, got %s", req.Header.Get("X-Amzn-Trace-Id"))
	}

	if req.Header.Get("X-Request-Id") != "test-request-id" {
		t.Errorf("expected X-Request-Id test-request-id, got %s", req.Header.Get("X-Request-Id"))
	}

	if req.Header.Get("X-Stage") != "test-stage" {
		t.Errorf("expected X-Stage test-stage, got %s", req.Header.Get("X-Stage"))
	}

	cookies := req.Header["Cookie"]
	if len(cookies) != 2 || cookies[0] != "cookie1=value1" || cookies[1] != "cookie2=value2" {
		t.Errorf("expected cookies [cookie1=value1, cookie2=value2], got %v", cookies)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}

	expectedBody := `{"key":"value"}`
	if string(body) != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, string(body))
	}
}

func TestConvertResponseV2_MultiValueHeaders(t *testing.T) {
	tests := []struct {
		name     string
		data     ResponseData
		expected events.APIGatewayV2HTTPResponse
	}{
		{
			name: "multi-value headers",
			data: ResponseData{
				StatusCode: http.StatusOK,
				Headers: http.Header{
					"Content-Type":    []string{"application/json"},
					"X-Custom-Header": []string{"value1", "value2"},
				},
				Body: []byte(`{"message":"Hello, World!"}`),
			},
			expected: events.APIGatewayV2HTTPResponse{
				StatusCode: http.StatusOK,
				Headers:    map[string]string{"Content-Type": "application/json"},
				MultiValueHeaders: map[string][]string{
					"X-Custom-Header": {"value1", "value2"},
				},
				Body: `{"message":"Hello, World!"}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := ConvertResponseV2(tt.data)
			if err != nil {
				t.Fatalf("ConvertResponseV2 failed: %v", err)
			}

			if resp.StatusCode != tt.expected.StatusCode {
				t.Errorf("expected status code %d, got %d", tt.expected.StatusCode, resp.StatusCode)
			}

			if resp.Body != tt.expected.Body {
				t.Errorf("expected body %q, got %q", tt.expected.Body, resp.Body)
			}

			for k, v := range tt.expected.Headers {
				if resp.Headers[k] != v {
					t.Errorf("expected header %s to be %q, got %q", k, v, resp.Headers[k])
				}
			}

			for k, v := range tt.expected.MultiValueHeaders {
				if !equalStringSlices(resp.MultiValueHeaders[k], v) {
					t.Errorf("expected multi-value header %s to be %v, got %v", k, v, resp.MultiValueHeaders[k])
				}
			}
		})
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
