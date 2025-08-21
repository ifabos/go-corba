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
- **Redis-based synchronization system for distributed instances**
- **Change listeners and data loaders for real-time synchronization**
- **Redis Streams and queues for message-based synchronization**

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

## Redis Synchronization

Go-CORBA now includes a Redis-based synchronization system that allows multiple instances to synchronize their state using Redis Streams and queues:

```go
package main

import (
    "context"
    "github.com/ifabos/go-corba/corba"
)

func main() {
    // Create sync configuration
    config := corba.DefaultSyncConfiguration()
    config.InstanceID = "instance1"
    config.RedisAddr = "localhost:6379"
    
    // Create synchronizer
    syncManager := corba.NewRedisStreamSynchronizer(config)
    
    // Start synchronization
    ctx := context.Background()
    if err := syncManager.Start(ctx); err != nil {
        panic(err)
    }
    defer syncManager.Stop()
    
    // Create ORB and synchronized event service
    orb := corba.Init()
    eventService := corba.NewSynchronizedEventService(orb, syncManager)
    
    // Operations will now be synchronized across instances
    channel, err := eventService.CreateChannel("test", corba.PushChannelType)
    // This will be synchronized to other instances
}
```

For detailed documentation on the Redis synchronization system, see [docs/REDIS_SYNC.md](docs/REDIS_SYNC.md).

## IDL Compiler

The Go-CORBA SDK includes an IDL compiler (`idlgen`) that translates CORBA IDL files to Go code.

### IDL Include Support

The IDL compiler fully supports the CORBA standard `#include` directive:

- System includes: `#include <file.idl>`
- User includes: `#include "file.idl"`

Include resolution follows the standard CORBA rules:
- User includes (`"file.idl"`) are first searched relative to the including file, then in the include path
- System includes (`<file.idl>`) are searched in the include path

```bash
# Example usage with include paths
idlgen -i myservice.idl -I /path/to/includes,/another/path -o ./generated
```

### Other IDL Features

The IDL compiler also supports:

- Repository IDs with `#pragma ID` directive
- Module prefixes with `#pragma prefix` directive
- Circular references detection and resolution
- Nested includes

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
