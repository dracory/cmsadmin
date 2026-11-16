# cmsadmin

Standalone CMS admin interface for [cmsstore](https://github.com/dracory/cmsstore).

Modeled on `github.com/dracory/shopadmin` and `github.com/dracory/blogadmin`:
the host mounts a single `http.Handler`, all navigation goes through
`?path=` query parameters, and pages render inside the host's layout via
`FuncLayout`.

```go
admin, err := cmsadmin.New(cmsadmin.AdminOptions{
    Store:  store,   // cmsstore.StoreInterface (required)
    Logger: logger,  // *slog.Logger (required)

    // Optional: wrap admin pages in your own layout
    FuncLayout: func(w http.ResponseWriter, r *http.Request, title, body string, options struct {
        Styles     []string
        StyleURLs  []string
        Scripts    []string
        ScriptURLs []string
    }) string {
        return myAdminLayout(title, body, options)
    },

    AdminHomeURL:    "/admin",
    MediaManagerURL: "/admin/media",
})

mux.Handle("/admin/cms", http.HandlerFunc(admin.Handle))
```

## License

AGPL-3.0. A commercial license is available — see LICENSE_COMMERCIAL.txt.
