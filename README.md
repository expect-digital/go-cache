# Go-Cache

Go-Cache is a high-performance, thread-safe library that provides a Least Recently Used (LRU) cache implementation in Go.

## Features

- **Generics**:  The cache leverages Go's generics to store any data type, providing flexibility and type safety.
- **Custom Getter**: The cache allows a custom getter function to be passed, which will be used to get the value if it is not present in the cache.
- **Eviction Callback:**: The cache allows a callback function to be passed, which will be called when an item is evicted from the cache.
- **LRU**: The cache evicts the least recently used items first when it reaches its size limit.
- **Customizable Size**: The size of the cache can be set according to the needs of the application.
- **Time-To-Live**: Each item in the cache has a TTL, after which it gets evicted.
- **Thread-Safe**: The cache operations are thread-safe, and can be safely used in concurrent applications.

## Getting Started

### Prerequisites

- Go version 1.18 or higher

### Installation

Install using `go get`
```sh
go get go.expect.digital/cache/lru
```

Import the `cache/lru` in your code
```go
import "go.expect.digital/cache/lru"
```

### Simple usage example
The following example shows all basic operations of the cache.
```go
type User struct {
  ID   int
  Name string
}

func basicExample(ctx context.Context) {
  userCache := lru.New[int, User]()

  user := User{ID: 1, Name: "John Doe"}

  // Set the user in the cache.
  err := userCache.Set(ctx, user.ID, user)
  if err != nil {
    // Handle error.
  }

  // Get the user from the cache.
  userFromCache, err := userCache.Get(ctx, user.ID)
  if err != nil {
    // Handle error.
  }

  fmt.Printf("Got user: %+v\n", userFromCache) // Got user: {ID:1 Name:John Doe}

  // Update cache entry.
  user.Name = "Jane Doe"

  err = userCache.Set(ctx, user.ID, user)
  if err != nil {
    // Handle error.
  }

  // Get updated user from the cache.
  userFromCache, err = userCache.Get(ctx, user.ID)
  if err != nil {
    // Handle error.
  }

  fmt.Printf("Got user: %+v\n", userFromCache) // Got user: {ID:1 Name:Jane Doe}

  // Trying to get a non-existing user from the cache will return an error.
  _, err = userCache.Get(ctx, 2) // err == lru.ErrNotFound

  // Get Size of cache
  fmt.Printf("Size of cache: %d\n", userCache.Size()) // Size of cache: 1024

  // Get count of stored values in cache
  fmt.Printf("Count of stored values in cache: %d\n", userCache.Len()) // Count of stored values in cache: 1
}
```

For more advanced examples, please, refer to the [package documentation](https://pkg.go.dev/go.expect.digital/cache/lru).

## License
Distributed under the MIT License. See [LICENSE](LICENSE) for more information.
