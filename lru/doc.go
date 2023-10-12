/*
Package lru implements a Least Recently Used (LRU) cache.

The cache is safe for concurrent access, which makes it suitable for use in multi-goroutine applications.

# Example Usage

## Basic

The following example shows all basic operations of the cache.

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

## Advanced

The following example shows how to use the cache with a getter function and a custom cache size.

	type User struct {
		ID   int
		Name string
	}


	// This function simulates getting a user from a database or other source.
	func getUser(_ context.Context, id int) (User, error) {
		// Simulate a delay for getting the user.
		time.Sleep(1 * time.Second)

		// Return a dummy user.
		return User{ID: id, Name: fmt.Sprintf("User %d", id)}, nil
	}

	func advancedExample(ctx context.Context) {
		userCache := lru.New[int, User](
			lru.WithGetter[int, User](getUser),
			lru.WithSize[int, User](2),
		)

		// With getter setting the user in the cache is not necessary.
		// If the user is not in the cache, it will be fetched by the getter.
		userFromCache, err := userCache.Get(ctx, 1)
		if err != nil {
			// Handle error.
		}

		fmt.Printf("Got user: %+v\n", userFromCache) // Got user: {ID:1 Name:User 1}

		// Get the user again. This should be fast because it's cached and the getter will not be called.
		userFromCache, err = userCache.Get(ctx, 1)
		if err != nil {
			// Handle error.
		}

		fmt.Printf("Got user: %+v\n", userFromCache) // Got user: {ID:1 Name:User 1}

		// Set another users in the cache.
		_, err = userCache.Get(ctx, 2)
		if err != nil {
			// Handle error.
		}

		// Setting another user in the cache will evict the least recently used user, which is user 1.
		_, err = userCache.Get(ctx, 3)
		if err != nil {
			// Handle error.
		}

		// Cache now contains user 2 and 3, as user 1 was evicted, because of the cache size of 2.
		// e.g.
		userCache.Get(ctx, 2) // Returns user 2 from the cache.
		userCache.Get(ctx, 3) // Returns user 3 from the cache.
		userCache.Get(ctx, 1) // Returns user 1 from the getter, sets it in the cache and evicts user 2.
	}
*/
package lru
