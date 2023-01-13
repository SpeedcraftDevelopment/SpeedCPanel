package main

import (
	"SpeedCPanelManager/schema"
	"context"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/james4k/rcon"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var rconClients map[string]context.Context = make(map[string]context.Context)
var rconCancels map[string]context.CancelFunc = make(map[string]context.CancelFunc)

type RCONStreamParams struct {
	Server       string `path:"service"`
	RCONPassword string `header:"X-RCON-Pass"`
	Command      string `header:"X-RCON-Cmd"`
}

func RCONExecuteCommand(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	if err := user.Claims.Valid(); err == nil {
		claims := user.Claims.(*jwtCustomClaims)
		var params RCONStreamParams
		var service schema.Container
		if err := c.Bind(&params); err != nil {
			return err
		}
		timeout, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		objectId, err := primitive.ObjectIDFromHex(params.Server)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		mongo := db.Database(config.DB.Database).Collection("Containers").FindOne(timeout, bson.M{"_id": objectId})
		err = mongo.Decode(&service)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		select {
		case <-rconClients[params.Server].Done():
			rconClient, err := rcon.Dial(service.Hostname, params.RCONPassword)
			if err != nil {
				if err == rcon.ErrAuthFailed {
					return echo.NewHTTPError(http.StatusUnauthorized, err)
				}
				return echo.NewHTTPError(http.StatusInternalServerError, err)
			}
			rconClients[params.Server], rconCancels[params.Server] = context.WithTimeout(ctx, time.Duration(config.Plans[claims.Plan].RCONTime)*time.Minute)
			rconClients[params.Server] = context.WithValue(rconClients[params.Server], params.Server, rconClient)
		}
		_, err = rconClients[params.Server].Value(params.Server).(*rcon.RemoteConsole).Write(params.Command)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		c.Response().WriteHeader(http.StatusOK)
		return nil
	} else {
		return err
	}
}

func RCONShutdown(c echo.Context) error {
	if err := rconClients[c.Param("service")].Value(c.Param("service")).(*rcon.RemoteConsole).Close(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}
	rconCancels[c.Param("service")]()
	c.Response().WriteHeader(http.StatusOK)
	return nil
}
