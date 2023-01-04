package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	logging "log"
	"os"
	"time"

	docker "github.com/docker/docker/client"
	jwt "github.com/golang-jwt/jwt/v4"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/pelletier/go-toml"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/labstack/echo/v4"
)

var (
	client *docker.Client
	config Config
	log    *logging.Logger
	db     *mongo.Client
	ctx    context.Context
)

type jwtCustomClaims struct {
	UserID   string `json:"id"`
	Username string `json:"name"`
	Plan     string `json:"subscription"`
	jwt.RegisteredClaims
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logFile, err := os.OpenFile(fmt.Sprintf("SpeedCPanel-%s-%s-%s-%s꞉%s꞉%s.log", fmt.Sprint(time.Now().Year()), time.Now().Month().String(), fmt.Sprint(time.Now().Day()), fmt.Sprint(time.Now().Hour()), fmt.Sprint(time.Now().Minute()), fmt.Sprint(time.Now().Second())), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	defer logFile.Close()
	if err != nil {
		panic(err)
	}
	log = logging.New(io.MultiWriter(os.Stdout, logFile), "SpeedPanel", logging.Default().Flags())
	configData, err := ioutil.ReadFile("config.toml")
	LogError(err)
	LogError(toml.Unmarshal(configData, &config))
	db, err = mongo.Connect(ctx, options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s@%s:%d/?tls=true", os.Getenv("MONGO_USERNAME"), os.Getenv("MONGO_PASSWORD"), config.DB.Hostname, config.DB.Port)))
	client, err = docker.NewClientWithOpts(docker.WithHost(config.Docker.Host),
		docker.WithVersion(config.Docker.Version),
		docker.WithTLSClientConfig(config.Docker.TLSConfig.CACertPath,
			config.Docker.TLSConfig.CertPath,
			config.Docker.TLSConfig.KeyPath),
		docker.WithHTTPHeaders(config.Docker.Headers))
	defer client.Close()
	LogError(err)
	server := echo.New()
	defer server.Close()
	jwtconfig := echojwt.Config{
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(jwtCustomClaims)
		},
		SigningKey: []byte(os.Getenv("JWTSecret")),
	}
	server.Use(echojwt.WithConfig(jwtconfig))
	server.POST("/api/v1/network", createNetwork)
}

func LogError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
