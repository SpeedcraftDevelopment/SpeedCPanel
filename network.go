package main

import (
	"SpeedCPanelManager/schema"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		timeout, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		createTraefikContainer, err := client.ServiceCreate(timeout, swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: fmt.Sprintf("%s_traefik", name),
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{
					Image:    config.Images["traefik"],
					TTY:      true,
					Hostname: fmt.Sprintf("%s-traefik", name),
					Mounts: []mount.Mount{
						{
							Source:   config.Traefik.ConfigPath,
							Target:   "/etc/traefik/traefik.toml",
							ReadOnly: true,
						},
						{
							Source:   "/var/run/docker.sock",
							Target:   "/var/run/docker.sock",
							ReadOnly: true,
						},
					},
				},
			},
			Networks: []swarm.NetworkAttachmentConfig{{Target: response.ID}},
		}, types.ServiceCreateOptions{})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		createNFSContainer, err := client.ServiceCreate(timeout, swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: fmt.Sprintf("%s_traefik", name),
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{
					Image:    config.Images["file_storage"],
					TTY:      true,
					Hostname: fmt.Sprintf("%s-nfs", name),
					Labels: map[string]string{
						"traefik.enable":                                      "true",
						"traefik.tcp.routers.nfs-router.rule":                 "HostSNI(`*`)",
						"traefik.tcp.routers.nfs-router.service":              "nfs-service",
						"traefik.tcp.services.nfs-service.loadbalancer.port":  "2049",
						"traefik.tcp.routers.sftp-router.rule":                "HostSNI(`*`)",
						"traefik.tcp.routers.sftp-router.service":             "sftp-service",
						"traefik.tcp.services.sftp-service.loadbalancer.port": "22",
					},
				},
			},
			Networks: []swarm.NetworkAttachmentConfig{{Target: response.ID}},
		}, types.ServiceCreateOptions{})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		teamowner := claims.TeamID != ""
		result, err := db.Database(config.DB.Database).Collection("Networks").InsertOne(timeout, schema.Network{
			OwnedByTeam: teamowner,
			Owner: func(team bool) string {
				if teamowner {
					return claims.TeamID
				} else {
					return claims.UserID
				}
			}(teamowner),
			Name:       name,
			DockerID:   response.ID,
			Containers: make([]int, 0),
			SpecialContainers: struct {
				Traefik string "bson:\"traefik\""
				NFS     string "bson:\"nfs\""
			}{
				Traefik: createTraefikContainer.ID,
				NFS:     createNFSContainer.ID,
			},
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
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
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		var res struct {
			types.NetworkCreateResponse `json:"network_resopnse"`
			MongoIDs                    struct {
				NetworkID string `json:"network_id"`
				OwnerID   string `json:"owner_id"`
				IsTeam    bool   `json:"team"`
			} `json:"database_response"`
		}
		res.NetworkCreateResponse = response
		res.MongoIDs.NetworkID = result.InsertedID.(primitive.ObjectID).Hex()
		res.MongoIDs.OwnerID = result2.UpsertedID.(primitive.ObjectID).Hex()
		res.MongoIDs.IsTeam = teamowner
		return c.JSON(http.StatusCreated, res)
	} else {
		return err
	}
}
