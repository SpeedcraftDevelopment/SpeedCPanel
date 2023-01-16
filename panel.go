package main

import (
	"context"
	"crypto/aes"
	"fmt"
	"io"
	"io/ioutil"
	logging "log"
	"net/http"
	"os"
	"time"

	docker "github.com/docker/docker/client"
	jwt "github.com/golang-jwt/jwt/v4"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/pelletier/go-toml"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	TeamID   string `json:"team"`
	Plan     string `json:"subscription"`
	jwt.RegisteredClaims
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	logFile, err := os.OpenFile(fmt.Sprintf("SpeedCPanel-%s-%s-%s-%s꞉%s꞉%s.log", fmt.Sprint(time.Now().Year()), time.Now().Month().String(), fmt.Sprint(time.Now().Day()), fmt.Sprint(time.Now().Hour()), fmt.Sprint(time.Now().Minute()), fmt.Sprint(time.Now().Second())), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	log = logging.New(io.MultiWriter(os.Stdout, logFile), "SpeedPanel", logging.Default().Flags())
	configData, err := ioutil.ReadFile("config.toml")
	LogError(err)
	LogError(toml.Unmarshal(configData, &config))
	db, err = mongo.Connect(ctx, options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s@%s:%d/?tls=true", os.Getenv("MONGO_USERNAME"), os.Getenv("MONGO_PASSWORD"), config.DB.Hostname, config.DB.Port)))
	LogError(err)
	defer func() {
		if err = db.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
	// Ping the primary
	if err := db.Ping(context.TODO(), readpref.Primary()); err != nil {
		log.Fatal(err)
	}
	client, err = docker.NewClientWithOpts(docker.WithHost(config.Docker.Host),
		docker.WithVersion(config.Docker.Version),
		docker.WithTLSClientConfig(config.Docker.TLSConfig.CACertPath,
			config.Docker.TLSConfig.CertPath,
			config.Docker.TLSConfig.KeyPath),
		docker.WithHTTPHeaders(config.Docker.Headers))
	LogError(err)
	defer client.Close()
	server := echo.New()
	defer server.Close()
	defer cancel()
	encryptedKey, err := ioutil.ReadFile(os.Getenv("PEMPath"))
	decryptedKey := make([]byte, len(encryptedKey))
	cipher, err := aes.NewCipher([]byte(os.Getenv("JWTSecret")))
	LogError(err)
	cipher.Decrypt(decryptedKey, encryptedKey)
	key, err := jwt.ParseECPublicKeyFromPEM(decryptedKey)
	LogError(err)
	jwtconfig := echojwt.Config{
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(jwtCustomClaims)
		},
		SigningMethod: jwt.SigningMethodES512.Name,
		SigningKey:    key,
	}
	server.Use(echojwt.WithConfig(jwtconfig))
	privateApiKeys := APIKey{config.ApiKeys}
	api := echo.New().Group("/api/v1", privateApiKeys.Process)
	api.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: config.AllowedURLs,
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete},
	}))
	api.POST("/network", createNetwork)
	api.POST("/:networkID/container", createContainer)
	api.PATCH("/:service", updateService)
	api.POST("/:service/console", RCONExecuteCommand)
	api.DELETE("/:service/console", RCONShutdown)
	api.GET("/:service/console", getLogs)
}

func LogError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
