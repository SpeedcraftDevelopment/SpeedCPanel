package main

import (
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/labstack/echo/v4"
)

func createNetwork(c echo.Context) error {
	name := c.Request().FormValue("name")
	response, err := client.NetworkCreate(c.Request().Context(), name, types.NetworkCreate{
		CheckDuplicate: true,
		Driver:         "overlay",
		Scope:          "swarm",
		EnableIPv6:     false,
		Internal:       false,
		Attachable:     true,
		Ingress:        false,
	})
	if err != nil {
		return err
	}
	var res struct {
		types.NetworkCreateResponse `json:"network_resopnse"`
	}
	res.NetworkCreateResponse = response
	return c.JSON(http.StatusAccepted, res)
}
