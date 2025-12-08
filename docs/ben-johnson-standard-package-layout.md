# Ben Johnson's Go standard package layout philosophy

Ben Johnson's approach to structuring Go applications has become one of the most influential architectural patterns in the Go community. His philosophy centers on a simple but powerful insight: packages in Go aren't groups—they're layers. This fundamental principle drives an entire system of organization that promotes clean architecture, testability, and maintainability as applications grow from simple utilities to complex distributed systems.

## The four core principles define the architecture

The Standard Package Layout rests on four interconnected principles that work together to create a scalable, maintainable architecture. **First, the root package contains only domain types**—pure business entities and interfaces that define what your application does without specifying how. **Second, subpackages are organized by dependency** rather than by function, with each external dependency getting its own package. **Third, a shared mock subpackage** provides simple, manual mocks for testing. **Fourth, the main package acts as the composition root**, wiring all dependencies together.

These principles address Go's unique constraint: the prohibition of circular dependencies. By treating packages as layers with dependencies flowing in one direction—always toward the domain—Johnson's approach eliminates the circular dependency problems that plague traditional MVC or module-based organizations.

## Domain types establish the foundation

The root package serves as the vocabulary of your application. Here live your core business entities, free from any external dependencies or implementation details. A User struct in the root package contains only the fields and methods that define what a user is in your business domain—no database tags, no JSON annotations beyond basic serialization needs, no references to external packages.

```go
package myapp

type User struct {
    ID        int
    Name      string
    Email     string
    CreatedAt time.Time
}

type UserService interface {
    FindUserByID(ctx context.Context, id int) (*User, error)
    CreateUser(ctx context.Context, user *User) error
    UpdateUser(ctx context.Context, id int, upd UserUpdate) (*User, error)
    DeleteUser(ctx context.Context, id int) error
}
```

**The interfaces defined here represent contracts, not implementations**. This inversion of control means that your domain defines what it needs, and implementations in subpackages fulfill those needs. The domain remains pure, testable, and comprehensible without understanding any infrastructure details.

## Dependencies live in isolation

Each external dependency gets its own package, named after the dependency it wraps. This seemingly controversial decision—having an `http` package in your application alongside Go's `net/http`—serves a critical purpose: it forces complete isolation of external dependencies.

```go
// postgres/user.go
package postgres

import (
    "context"
    "database/sql"
    "github.com/benbjohnson/myapp"
)

type UserService struct {
    db *sql.DB
}

func (s *UserService) FindUserByID(ctx context.Context, id int) (*myapp.User, error) {
    var u myapp.User
    err := s.db.QueryRowContext(ctx, 
        `SELECT id, name, email, created_at FROM users WHERE id = $1`, 
        id).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
    if err == sql.ErrNoRows {
        return nil, &myapp.Error{Code: myapp.ENOTFOUND, Message: "user not found"}
    }
    return &u, err
}
```

This structure enables powerful architectural patterns. You can layer implementations—wrapping a PostgreSQL service with a Redis caching layer, both implementing the same interface. You can swap implementations entirely, moving from PostgreSQL to MySQL by changing only the main package wiring. Most importantly, you can test each layer in complete isolation.

## Testing becomes trivial with shared mocks

Rather than relying on code generation or complex mocking frameworks, Johnson advocates for simple, manual mocks in a shared package. This approach provides complete control over test behavior while remaining easy to understand and modify.

```go
// mock/user.go
package mock

type UserService struct {
    FindUserByIDFn func(ctx context.Context, id int) (*myapp.User, error)
    FindUserByIDInvoked bool
}

func (m *UserService) FindUserByID(ctx context.Context, id int) (*myapp.User, error) {
    m.FindUserByIDInvoked = true
    return m.FindUserByIDFn(ctx, id)
}
```

These mocks enable focused unit testing of any component without setting up databases, external services, or complex test harnesses. Each test specifies exactly what behavior it needs from dependencies, making tests fast, reliable, and easy to understand.

## The main package orchestrates everything

Dependency injection happens at compile time in the main package. This composition root knows about all implementations and wires them together, but the wiring code remains simple and obvious.

```go
// cmd/myapp/main.go
package main

import (
    "github.com/benbjohnson/myapp"
    "github.com/benbjohnson/myapp/postgres"
    "github.com/benbjohnson/myapp/http"
)

func main() {
    // Initialize database
    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create services
    userService := &postgres.UserService{DB: db}
    
    // Create HTTP server with injected dependencies
    server := &http.Server{
        UserService: userService,
    }
    
    // Start server
    log.Fatal(server.ListenAndServe(":8080"))
}
```

**This explicit wiring provides several benefits**: dependencies are visible and traceable, there's no magic or reflection, and the entire dependency graph is clear at compile time. The main package typically remains small since it only handles wiring and configuration.

## Error handling follows domain principles

Johnson's approach extends to error handling, with domain-specific error types that convey both machine-readable codes and human-readable messages. This structured approach works particularly well for APIs and services that need to communicate errors clearly.

```go
// error.go in root package
const (
    ECONFLICT     = "conflict"
    EINTERNAL     = "internal"
    EINVALID      = "invalid"
    ENOTFOUND     = "not_found"
    EUNAUTHORIZED = "unauthorized"
)

type Error struct {
    Code    string
    Message string
}

func (e *Error) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
```

Infrastructure packages translate their specific errors into these domain errors, maintaining the abstraction boundary while providing useful error information throughout the stack.

## The approach scales with your application

For applications under 10,000 lines, Johnson acknowledges that a single package might suffice. The Standard Package Layout shines as applications grow beyond this threshold, providing structure that supports team collaboration, feature addition, and long-term maintenance.

**The WTF Dial project demonstrates this scalability**. Starting as a simple real-time dashboard, it showcases how the architecture handles authentication, real-time events, database migrations, and HTTP APIs while maintaining clear separation between concerns. Each feature slots naturally into the existing structure without requiring architectural changes.

## Common pitfalls and how to avoid them

Teams implementing this approach often encounter similar challenges. **Domain modeling proves difficult initially**—the temptation to include implementation details in domain types remains strong. Resist this by asking whether each field or method represents business logic or infrastructure concern.

**Package boundaries blur without discipline**. The solution lies in the interface definitions: if package A needs functionality from package B, define an interface in package A that package B can implement. Dependencies flow toward the domain, never between infrastructure packages.

**Over-engineering threatens smaller projects**. Start simple—even with a single package—and extract packages as natural boundaries emerge. The structure should serve the code, not constrain it.

## Evolution through practical application

Johnson's philosophy has evolved through community feedback and real-world application. The initial four tenets remain, but the understanding has deepened. **Packages as layers, not groups** became the clarifying insight that explains why this approach works where others fail in Go.

The community has adapted the pattern to various contexts—microservices extract individual dependency packages into separate services, event-driven architectures add event packages following the same patterns, and CLI applications use the structure for command organization. The core principles prove remarkably flexible.

## Real-world benefits: The OpenAI to Vertex AI migration

The power of this architecture became evident during our migration from OpenAI to Google Vertex AI. This real-world example demonstrates how Johnson's principles enable clean dependency swapping with minimal code changes.

### The interface enabled seamless migration

The domain defined what it needed through a simple interface:

```go
// In bookscanner/scanner.go
type BookParser interface {
    ExtractSearchQuery(ctx context.Context, ocrText string) (string, error)
    ParseBookInfo(ctx context.Context, text string) (*Book, error)
}
```

Both OpenAI and Vertex AI implementations satisfied this contract in isolation:

```go
// scanner/openai/openai.go
type Client struct {
    openaiClient *openai.Client
}

// scanner/vertexai/vertexai.go  
type Client struct {
    genaiClient *genai.Client
    modelName   string
}
```

The migration required changing only the dependency injection:

```go
// Before
bookParser := openai.New(apiKey)

// After  
bookParser := vertexai.New(ctx, projectID, location)
```

### Key learnings from the migration

1. **No ripple effects** - The interface boundary contained all changes
2. **Tests remained valid** - Behavioral tests worked for both implementations
3. **Compile-time safety** - Interface compliance caught issues immediately
4. **Risk minimization** - We could run both providers in parallel during migration

This migration validated that with proper interface design, dependencies truly become swappable. The architecture didn't just promise flexibility—it delivered it when we needed it most.

## Conclusion

Ben Johnson's Standard Package Layout offers more than a directory structure—it provides a philosophy for thinking about Go applications as layered systems with clear dependencies and boundaries. By embracing Go's constraints rather than fighting them, this approach creates applications that are testable, maintainable, and scalable.

The pattern's success lies not in rigid adherence to rules but in understanding the principles behind them. Start with domain types, isolate dependencies, test in isolation, and wire explicitly. These simple guidelines, applied consistently, lead to Go applications that remain comprehensible and modifiable as they grow from simple utilities to complex distributed systems—as our OpenAI to Vertex AI migration clearly demonstrated.