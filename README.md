# Gateway HTTP ListenAndServe for AWS Lambda (V1 and V2 API Gateway)

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE-OF-CONDUCT.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
![GitHub release](https://img.shields.io/github/release/go-obvious/gateway.svg)
![Status](https://img.shields.io/badge/status-stable-green.svg)

This Go library provides a drop-in replacement for `http.ListenAndServe` that is optimized for AWS Lambda's API Gateway (both V1 and V2). The library abstracts away the differences between API Gateway V1 and V2, allowing you to focus on building your application without worrying about the underlying details.

## Features

- Supports both API Gateway V1 (`APIGatewayProxyRequest`/`APIGatewayProxyResponse`) and V2 (`APIGatewayV2HTTPRequest`/`APIGatewayV2HTTPResponse`).
- Simple, familiar interface that works with existing `http.Handler` code.
- Lightweight, with the underlying complexity hidden.

## Getting Started

### Installation

1. Install the library using Go modules:

    ```bash
    go get github.com/go-obvious/gateway
    ```

2. Import the package in your Go project:

    ```go
    import "github.com/go-obvious/gateway"
    ```

### Usage

The library provides two simple entry points for API Gateway V1 and V2:

* `ListenAndServeV1`: For API Gateway V1
* `ListenAndServeV2`: For API Gateway V2

### Example: API Gateway V1

For API Gateway V1, use `ListenAndServeV1` as a replacement for `http.ListenAndServe`.

```go
package main

import (
    "github.com/yourusername/gateway"
    "net/http"
    "encoding/json"
)

func myHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"message": "Hello from V1 API Gateway!"})
}

func main() {
    // Use ListenAndServeV1 for API Gateway V1
    gateway.ListenAndServeV1(":8080", http.HandlerFunc(myHandler))
}
```

### Example: API Gateway V2

For API Gateway V2, use `ListenAndServeV2` as a replacement for `http.ListenAndServe`.

```go
package main

import (
    "github.com/yourusername/gateway"
    "net/http"
    "encoding/json"
)

func myHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"message": "Hello from V2 API Gateway!"})
}

func main() {
    // Use ListenAndServeV2 for API Gateway V2
    gateway.ListenAndServeV2(":8080", http.HandlerFunc(myHandler))
}
```

### How It Works

- **ListenAndServeV1**: Automatically parses and handles API Gateway V1 requests.
- **ListenAndServeV2**: Automatically parses and handles API Gateway V2 requests.
- Both versions use the familiar `http.Handler` interface, making it easy to port existing HTTP applications to AWS Lambda.

### FAQ

#### 1. Can I use this library outside of AWS Lambda?

No, this library is specifically designed to work with API Gateway in AWS Lambda environments.

#### 2. Can I use both API Gateway versions in the same application?

Yes, you can use both `ListenAndServeV1` and `ListenAndServeV2` within the same application, depending on which API Gateway version you are targeting.

## Contributing

Feel free to submit issues or pull requests for new features, bug fixes, or improvements.

## License

This library is licensed under the MIT License. See `LICENSE` for more details.
