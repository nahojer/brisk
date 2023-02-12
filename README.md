# brisk

Fast and simple HTTP router for Go applications. Supports

* path parameterization,
* middleware,
* prefix route matching, and
* named sub-routers (route groups).


Uses [sage](https://github.com/nahojer/sage) under the hood to match incoming HTTP requests to handlers.

All of the documentation can be found on the [go.dev](https://pkg.go.dev/github.com/nahojer/brisk?tab=doc) website.

Is it Good? [Yes](https://news.ycombinator.com/item?id=3067434).

## Example Usage

Below example depicts a real-worl usecase of `brisk` in a REST API. There is assumed to be an 
existing `mid` package containing the set of middleware functions, a `auth` package 
exporting the valid authorization roles a user can take, and a `user` package managing the
user business logic.

```go
shutdown := make(chan os.Signal, 1) // Channel to signal graceful shutdown of the HTTP server 

// ...

router := brisk.NewRouter(mid.RecoverPanics(), mid.CORS(corsOrigin), mid.Logger(log))
router.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
  // If we got this far it means an unexpected error occurred and that we should 
  // shutdown the HTTP server. All expected errors are handled by the mid.Errors 
  // middleware set on the api/v1 group defined below.
  if isShutdown(err) {
    shutdown <- syscall.SIGTERM
  }
}

// CORS
{
  h := func(_ http.ResponseWriter, _ *http.Request) error { 
    return nil 
  }
  router.Handle(http.MethodOptions, "/...", h)
}

// API version 1 routes
{
  v1 := router.Group("api/v1", mid.Errors(log))
  {
    uh := v1UserHandlers{
      User: user.NewCore(db),
    }
    
    users := v1.Group("users", mid.Authenticate())
    users.Get("/", uh.Query, mid.Authorize(auth.RoleAdmin))
    users.Get("/:id", uh.QueryByID, mid.Authorize(auth.RoleSubject, auth.RoleAdmin)) 
    users.Post("/", uh.Create, mid.Authorize(auth.RoleAdmin))
    users.Put("/:id", uh.Update, mid.Authorize(auth.RoleSubject, auth.RoleAdmin))
    // The final path pattern for below handler is /api/v1/users/:id.
    users.Delete("/:id", uh.Delete, mid.Authorize(auth.RoleSubject, auth.RoleAdmin))
  }
}

// ...

type v1UserHandlers struct {
  User user.Core
}

func (uh v1UserHandlers) QueryByID(w http.ResponseWriter, r *http.Request) error {
  id := brisk.Param(r, "id")

  user, err := uh.User.Query(r.Context(), id)
  if err != nil {
    return err
  }

  jsonData, err := json.Marshal(&user)
  if err != nil {
    return err
  }

  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(http.StatusOK)

  if _, err := w.Write(jsonData); err != nil {
    return err
  }

  return nil
}

// ...
```
