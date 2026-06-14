// Package docs serves the OpenAPI spec and a Swagger UI page that renders it.
//
// The spec bytes are owned by the api package (embedded next to openapi.yaml).
// This package is just the HTTP adapter — two handlers, no state.
package docs

import (
	"net/http"

	"github.com/gab-mello/click-and-collect/api"
)

// Swagger UI assets are pinned to a specific version so the docs page is
// reproducible and isn't at the mercy of upstream breakage. Bump deliberately.
const swaggerUIVersion = "5.17.14"

// SpecHandler serves the embedded OpenAPI YAML.
func SpecHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		// The spec is embedded at build time; if it changes, the binary
		// changes — but during local development we'd rather not serve a
		// stale browser cache.
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(api.OpenAPISpec)
	})
}

// UIHandler serves a single static HTML page that loads Swagger UI from a
// pinned jsDelivr URL and points it at /openapi.yaml.
func UIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(uiHTML))
	})
}

// uiHTML is the Swagger UI bootstrap page. Kept as a constant — a template
// would be overkill for two interpolated values that almost never change.
const uiHTML = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>Click & Collect API – Docs</title>
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <link
      rel="stylesheet"
      href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@` + swaggerUIVersion + `/swagger-ui.css"
    />
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@` + swaggerUIVersion + `/swagger-ui-bundle.js" crossorigin></script>
    <script>
      window.addEventListener("load", function () {
        window.ui = SwaggerUIBundle({
          url: "/openapi.yaml",
          dom_id: "#swagger-ui",
          deepLinking: true,
          presets: [SwaggerUIBundle.presets.apis],
          layout: "BaseLayout",
        });
      });
    </script>
  </body>
</html>
`
