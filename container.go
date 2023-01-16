package main

import (
	"SpeedCPanelManager/schema"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ServerCreateParams struct {
	Name      string `json:"name"`
	Hostname  string `json:"hostname"`
	NetworkId int    `path:"networkID"`
	Version   string `json:"version"`
	Image     string `head:"X-Docker-Image"`
	Type      string `json:"type"`
	Premium   bool   `json:"premium"`
	Modpack   string `json:"modpack,omitempty"`
}

func createContainer(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	if err := user.Claims.Valid(); err == nil {
		claims := user.Claims.(*jwtCustomClaims)
		var params ServerCreateParams
		c.Bind(&params)
		var network schema.Network
		if err != nil {
			return err
		}
		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		res := db.Database(config.DB.Database).Collection("Networks").FindOne(timeoutCtx, bson.D{{
			"_id",
			params.NetworkId,
		}})
		if err = res.Decode(network); err != nil {
			return err
		}
		env := []string{"EULA=TRUE", fmt.Sprintf("VERSION=%s", params.Version, fmt.Sprintf("TYPE=%s", params.Type))}
		if params.Type == "CURSEFORGE" {
			env = append(env, fmt.Sprintf("CF_SERVER_MOD=%s", params.Modpack))
		} else if params.Type == "FTBA" {
			env = append(env, fmt.Sprintf("FTB_MODPACK_ID=%s", params.Modpack))
		}
		result, err := client.ServiceCreate(timeoutCtx, swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: params.Name,
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{
					Image:    config.Images[params.Image],
					Env:      env,
					Hostname: params.Hostname,
					Labels: map[string]string{
						"traefik.tcp.routers.mc.rule": "HostSNI(`*`)",
						"traefik.port":                strconv.Itoa(25565 + len(network.Containers)),
					},
				},
			},
			Networks: []swarm.NetworkAttachmentConfig{{Target: network.DockerID}},
		}, types.ServiceCreateOptions{})
		if err != nil {
			return err
		}
		mongoresult, err := db.Database(config.DB.Database).Collection("Containers").InsertOne(timeoutCtx, schema.Container{
			DockerID: result.ID,
			Name:     params.Name,
			Image:    config.Images[params.Image],
			Hostname: params.Hostname,
			Owner: func(isTeam bool) string {
				if isTeam {
					return claims.TeamID
				} else {
					return claims.UserID
				}
			}(claims.TeamID != ""),
			IsOwnerTeam: claims.TeamID != "",
		})
		if err != nil {
			return err
		}
		netupdateresult, err := db.Database(config.DB.Database).Collection("Networks").UpdateByID(timeoutCtx, params.NetworkId, bson.D{{
			"$addToSet",
			bson.D{{
				"Containers",
				mongoresult.InsertedID,
			}},
		}})
		var finalresult struct {
			types.ServiceCreateResponse `json:"server_create"`
			InsertOneResult             struct {
				InsertedID string
			} `json:"server_insert_result"`
			UpdateResult struct {
				MatchedCount  int64
				ModifiedCount int64
				UpsertedCount int64
				UpsertedID    string
			} `json:"network_update_result"`
		}
		finalresult.ServiceCreateResponse = result
		finalresult.InsertOneResult.InsertedID = mongoresult.InsertedID.(primitive.ObjectID).Hex()
		finalresult.UpdateResult.MatchedCount = netupdateresult.MatchedCount
		finalresult.UpdateResult.ModifiedCount = netupdateresult.ModifiedCount
		finalresult.UpdateResult.UpsertedCount = netupdateresult.UpsertedCount
		finalresult.UpdateResult.UpsertedID = netupdateresult.UpsertedID.(primitive.ObjectID).Hex()
		return c.JSON(http.StatusCreated, finalresult)
	} else {
		return err
	}
}

type ServerUpdateParams struct {
	Server    string `path:"service"`
	NetworkId string `header:"Network-ID,omitempty"`
	Version   string `header:"Version,omitempty"`
	Port      int    `header:"Port,omitempty"`
}

func updateService(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	if err := user.Claims.Valid(); err == nil {
		var params ServerUpdateParams
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
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Errorf("Attempted unauthorised update of container."))
		}
		servicedata, err := client.ServiceList(timeout, types.ServiceListOptions{
			Filters: filters.NewArgs(filters.KeyValuePair{"id", cont.DockerID}),
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		if params.NetworkId != "" {
			var netwotk schema.Network
			mongo = db.Database(config.DB.Database).Collection("Networks").FindOne(timeout, bson.M{"_id": params.Server})
			if err = mongo.Decode(netwotk); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err)
			}
			servicedata[0].Spec.Networks[0] = swarm.NetworkAttachmentConfig{Target: netwotk.DockerID}
		}
		if params.Version != "" {
			env := servicedata[0].Spec.TaskTemplate.ContainerSpec.Env
			re := regexp.MustCompile(`^VERSION=([0-9]+\.[0-9]+(\.[0-9]+)?)$`)
			for i, str := range env {
				if re.Match([]byte(str)) {
					env[i] = "VERSION=" + params.Version
				}
			}
		}
		if params.Port != 0 {
			servicedata[0].Spec.TaskTemplate.ContainerSpec.Labels["traefik.port"] = strconv.Itoa(params.Port)
		}
		updateResult, err := client.ServiceUpdate(timeout, cont.DockerID, servicedata[0].Version, servicedata[0].Spec, types.ServiceUpdateOptions{})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		return c.JSON(http.StatusOK, updateResult)
	} else {
		return err
	}
}
