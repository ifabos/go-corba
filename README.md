# Go-CORBA

A Go implementation of the Common Object Request Broker Architecture (CORBA) specification.

## Overview

Go-CORBA provides a complete Software Development Kit (SDK) for building CORBA applications in Go. It enables developers to create distributed systems where objects written in Go can interact with objects written in other programming languages that implement the CORBA standard.

## Features

- CORBA IDL (Interface Definition Language) support
- ORB (Object Request Broker) implementation
- GIOP/IIOP protocol support
- Naming Service integration
- Dynamic Invocation Interface (DII)
- Type management and marshaling

## Installation

```bash
go get github.com/ifabos/go-corba
```

## Usage

Basic example of creating a CORBA server:

```go
package main

import (
    "github.com/ifabos/go-corba/orb"
    "github.com/ifabos/go-corba/naming"
)

func main() {
    // Initialize the ORB
    myORB := orb.Init()
    
    // Create a server instance
    server := myORB.CreateServer()
    
    // Register an object
    // ...
    
    // Run the server
    server.Run()
}
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
