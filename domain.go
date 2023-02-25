package main

import (
	"SpeedCPanelManager/schema"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cloudflare/cloudflare-go"
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
		var srv struct {
			Prioriry   int    `json:"priority"`
			Weight     int    `json:"weight"`
			Port       int    `json:"port"`
			Target     string `json:"target"`
			Service    string `json:"service"`
			Protocol   string `json:"proto"`
			DomainName string `json:"name"`
		}
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
		zone, err := cfApi.CreateZone(timeout, params.Domain, true, cfAcc, "partial")
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		recordResponse1, err := cfApi.CreateDNSRecord(timeout, cloudflare.ZoneIdentifier(zone.ID), cloudflare.CreateDNSRecordParams{
			Name:      "@",
			Content:   "W.I.P", //Muszę się nauczyć jak znaleśc adres IP usługi Dockera
			TTL:       config.Plans[claims.UserID].TTL,
			Type:      "A",
			Proxiable: false,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		recordResponse2, err := cfApi.CreateDNSRecord(timeout, cloudflare.ZoneIdentifier(zone.ID), cloudflare.CreateDNSRecordParams{
			Name:    fmt.Sprintf("_minecraft._tcp.%s", params.Domain),
			Content: fmt.Sprintf("SRV 0 0 %d %s", cont.Port, params.Domain),
			TTL:     config.Plans[claims.UserID].TTL,
			Type:    "SRV",
			Data:    srv,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		var response struct {
			ZoneID    string `json:"zone"`
			Record1ID string `json:"a-record"`
			Record2ID string `json:"srv-record"`
		}
		response.ZoneID = zone.ID
		response.Record1ID = recordResponse1.Result.ID
		response.Record2ID = recordResponse2.Result.ID
		return c.JSON(http.StatusAccepted, response)
	} else {
		return err
	}
}
