package swagger

import (
	"embed"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed openapi.yaml
var specFS embed.FS

// RegisterRoutes adds Swagger UI and spec endpoints to the router.
// GET /swagger/*path serves both the UI and the spec file.
func RegisterRoutes(r *gin.Engine) {
	r.GET("/swagger/*path", func(c *gin.Context) {
		path := c.Param("path")
		if strings.HasSuffix(path, "openapi.yaml") {
			data, err := specFS.ReadFile("openapi.yaml")
			if err != nil {
				c.String(http.StatusInternalServerError, "spec not found")
				return
			}
			c.Data(http.StatusOK, "application/yaml", data)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUIHTML()))
	})
}

func swaggerUIHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Ride-Hailing API - Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>html{box-sizing:border-box;overflow-y:scroll}*,*:before,*:after{box-sizing:inherit}body{margin:0;background:#fafafa}</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: '/swagger/openapi.yaml',
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: "BaseLayout",
      deepLinking: true,
    });
  </script>
</body>
</html>`
}
