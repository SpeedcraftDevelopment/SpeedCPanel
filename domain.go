package main

import (
	"SpeedCPanelManager/schema"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
)

type (
	DomainSetParameters struct {
		Server string `path:"service"`
		Domain string `head:"domain"`
	}
)

func setDomain(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	if err := user.Claims.Valid(); err == nil {
		var params DomainSetParameters
		var cont schema.Container
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
	} else {
		return err
	}
}
