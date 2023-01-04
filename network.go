package main

import (
	"SpeedCPanelManager/schema"
	"context"
	"net/http"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
)

func createNetwork(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	if err := user.Claims.Valid(); err == nil {
		claims := user.Claims.(*jwtCustomClaims)
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
		timeout, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		result, err := db.Database(config.DB.Database).Collection("Users").UpdateByID(timeout, claims.UserID, bson.D{{"$addToSet", bson.D{{"networks", schema.Network{
			Name:       name,
			Containers: make([]schema.Container, 0),
		}}}}})
		var res struct {
			types.NetworkCreateResponse `json:"network_resopnse"`
			ID                          int `json:"database_response"`
		}
		res.NetworkCreateResponse = response
		res.ID = result.UpsertedID.(int)
		return c.JSON(http.StatusAccepted, res)
	} else {
		return err
	}
}
