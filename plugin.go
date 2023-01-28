package main

import (
	"SpeedCPanelManager/schema"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
)

type AddPluginParams struct {
	Server   string `path:"service"`
	PluginId string `head:"X-Plugin-ID"`
}

func addPlugin(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	if err := user.Claims.Valid(); err == nil {
		var params AddPluginParams
		var cont schema.Container
		hasSpigetVariable := false
		claims := user.Claims.(*jwtCustomClaims)
		c.Bind(&params)
		timeout, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		mongo := db.Database(config.DB.Database).Collection("Containers").FindOne(timeout, bson.M{"_id": params.Server})
		if err = mongo.Decode(cont); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		if claims.UserID != cont.Owner || claims.TeamID != cont.Owner {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Errorf("attempted unauthorised update of container"))
		}
		servicedata, err := client.ServiceList(timeout, types.ServiceListOptions{
			Filters: filters.NewArgs(filters.KeyValuePair{"id", cont.DockerID}),
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		re := regexp.MustCompile(`^(SPIGET_RESOURCES=)(([0-9]+)\,\s*)*(?P<pluginId>[0-9]+)$`)
		for index, env := range servicedata[0].Spec.TaskTemplate.ContainerSpec.Env {
			if re.MatchString(env) {
				hasSpigetVariable = true
				if contains(strings.Split(strings.Trim(env, "SPIGET_ROUC="), ","), compareString(params.PluginId)) {
					return echo.NewHTTPError(http.StatusAlreadyReported, fmt.Errorf("plugin already present"))
				} else {
					servicedata[0].Spec.TaskTemplate.ContainerSpec.Env[index] += ("," + params.PluginId)
				}
			}
		}
		if !hasSpigetVariable {
			servicedata[0].Spec.TaskTemplate.ContainerSpec.Env = append(servicedata[0].Spec.TaskTemplate.ContainerSpec.Env, fmt.Sprintf("SPIGET_RESOURCES=%s", params.PluginId))
		}
		response, err := client.ServiceUpdate(timeout, params.Server, servicedata[0].Version, servicedata[0].Spec, types.ServiceUpdateOptions{})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		return c.JSON(http.StatusOK, response)
	} else {
		return err
	}
}
