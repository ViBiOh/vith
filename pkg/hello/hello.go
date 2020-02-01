package hello

import (
	"flag"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/ViBiOh/httputils/v3/pkg/flags"
	"github.com/ViBiOh/httputils/v3/pkg/httpjson"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/httputils/v3/pkg/swagger"
)

// Hello represents the outputted welcome message
type Hello struct {
	Name string `json:"greeting"`
}

// Config of package
type Config struct {
	locationName *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		locationName: flags.New(prefix, "hello").Name("Location").Default("Europe/Paris").Label("TimeZone for displaying current time").ToString(fs),
	}
}

// Handler for Hello request. Should be use with net/http
func Handler(config Config) http.Handler {
	location, err := time.LoadLocation(*config.locationName)
	if err != nil {
		logger.Error("error while loading location %s: %v", *config.locationName, err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		name := strings.TrimPrefix(html.EscapeString(r.URL.Path), "/")
		if name == "" {
			name = "World"
		}

		httpjson.ResponseJSON(w, http.StatusOK, Hello{fmt.Sprintf("Hello %s, current time in %s is %v !", name, location.String(), time.Now().In(location))}, httpjson.IsPretty(r))
	})
}

// Swagger provides swagger configuration for
func Swagger() (swagger.Configuration, error) {
	return swagger.Configuration{
		Paths: `/hello/{name}:
  get:
    description: Say hello
    parameters:
      - name: name
        in: path
        description: Name to greet
        required: false
        schema:
          type: string

    responses:
      200:
        description: Greeting message
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Hello'`,
		Components: `Hello:
  type: object
  properties:
    greeting:
      type: string
      description: Greeting sentence`,
	}, nil
}
