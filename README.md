# AWS Gateway

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE-OF-CONDUCT.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
![GitHub release](https://img.shields.io/github/release/go-obvious/gateway.svg)
![Status](https://img.shields.io/badge/status-stable-green.svg)

AWS Gateway provides a drop-in replacement for net/http's `ListenAndServe` for use in [AWS Lambda](https://aws.amazon.com/lambda/) & [API Gateway](https://aws.amazon.com/api-gateway/). Simply swap it out for `gateway.ListenAndServe`. This library is extracted from [Up](https://github.com/apex/up), which provides additional middleware features and operational functionality.

There are two versions of this library:
- **Version 1.x**: Supports AWS API Gateway 1.0 events used by the original [REST APIs](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-rest-api.html).
- **Version 2.x**: Supports 2.0 events used by the [HTTP APIs](https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api.html).

For more information on the options, read [Choosing between HTTP APIs and REST APIs](https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api-vs-rest.html) on the AWS documentation website.

## Installation

To install version 1.x for REST APIs:

```sh
go get github.com/go-obvious/gateway
```

To install version 2.x for HTTP APIs:

```sh
go get github.com/go-obvious/gateway/v2
```

## Example

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-obvious/gateway"
)

func main() {
	http.HandleFunc("/", hello)
	log.Fatal(gateway.ListenAndServe(":3000", nil))
}

func hello(w http.ResponseWriter, r *http.Request) {
	// Example retrieving values from the API Gateway proxy request context.
	requestContext, ok := gateway.RequestContext(r.Context())
	if !ok || requestContext.Authorizer["sub"] == nil {
		fmt.Fprint(w, "Hello World from Go")
		return
	}

	userID := requestContext.Authorizer["sub"].(string)
	fmt.Fprintf(w, "Hello %s from Go", userID)
}
```
