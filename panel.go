package main

import (
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

	"github.com/labstack/echo/v4"
)

var (
	client *docker.Client
	config Config
	log    *logging.Logger
)

type jwtCustomClaims struct {
	UserID   string `json:"id"`
	Username string `json:"name"`
	Plan     string `json:"subscription"`
	jwt.RegisteredClaims
}

func main() {
	logFile, err := os.OpenFile(fmt.Sprintf("SpeedCPanel-%s-%s-%s-%s꞉%s꞉%s.log", fmt.Sprint(time.Now().Year()), time.Now().Month().String(), fmt.Sprint(time.Now().Day()), fmt.Sprint(time.Now().Hour()), fmt.Sprint(time.Now().Minute()), fmt.Sprint(time.Now().Second())), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	log = logging.New(io.MultiWriter(os.Stdout, logFile), "SpeedPanel", logging.Default().Flags())
	configData, err := ioutil.ReadFile("config.toml")
	LogError(err)
	LogError(toml.Unmarshal(configData, &config))
	client, err = docker.NewClientWithOpts(docker.FromEnv)
	LogError(err)
	server := echo.New()
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
