package main

import (
	"SpeedCPanelManager/schema"
	"context"
	"encoding/binary"
	"net/http"
	"time"

	"github.com/docker/docker/api/types"
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

func getLogs(c echo.Context) error {
	c.Response().Header().Set("Connection", "Keep-Alive")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")
	c.Response().Header().Set("Content-Type", "text/event-stream")
	reader, err := client.ServiceLogs(ctx, c.Param("service"), types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
		Tail:       "40",
		Details:    false,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}
	defer func() {
		if err = reader.Close(); err != nil {
			panic(err)
		}
	}()
	hdr := make([]byte, 8)
	for {
		if _, err = reader.Read(hdr); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		count := binary.BigEndian.Uint32(hdr[4:])
		dat := make([]byte, count)
		if _, err = reader.Read(dat); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		c.Response().Write(dat)
		c.Response().Flush()
	}

}
