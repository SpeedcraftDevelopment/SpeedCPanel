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
	"go.mongodb.org/mongo-driver/mongo"
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
		teamowner := claims.TeamID != -1
		result, err := db.Database(config.DB.Database).Collection("Networks").InsertOne(timeout, schema.Network{
			OwnedByTeam: teamowner,
			Owner: func(team bool) int {
				if teamowner {
					return claims.TeamID
				} else {
					return claims.UserID
				}
			}(teamowner),
			Name:       name,
			DockerID:   response.ID,
			Containers: make([]schema.Container, 0),
		})
		var result2 *mongo.UpdateResult
		result2, err = db.Database(config.DB.Database).Collection(func(team bool) string {
			if teamowner {
				return "Teams"
			} else {
				return "Users"
			}
		}(teamowner)).UpdateByID(ctx, claims.TeamID, bson.D{{
			"$addToSet",
			bson.D{{
				"networks",
				result.InsertedID.(int),
			}},
		}})
		if err != nil {
			return err
		}
		var res struct {
			types.NetworkCreateResponse `json:"network_resopnse"`
			MongoIDs                    struct {
				NetworkID int  `json:"network_id"`
				OwnerID   int  `json:"owner_id"`
				IsTeam    bool `json:"team"`
			} `json:"database_response"`
		}
		res.NetworkCreateResponse = response
		res.MongoIDs.NetworkID = result.InsertedID.(int)
		res.MongoIDs.OwnerID = result2.UpsertedID.(int)
		res.MongoIDs.IsTeam = teamowner
		return c.JSON(http.StatusAccepted, res)
	} else {
		return err
	}
}
