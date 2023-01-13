package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

type APIKey struct {
	APIKeys []string
}

func (key *APIKey) Process(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		apiKey := c.Get("X-API-Key").(string)
		if !contains(key.APIKeys, compareString(apiKey)) {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("API Key %s is not legitimate", apiKey))
		} else {
			return nil
		}
	}
}
