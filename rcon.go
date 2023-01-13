package main

import (
	"net/http"
	"strings"

	"github.com/james4k/rcon"
	"github.com/labstack/echo/v4"
)

type RCONStreamParams struct {
	RCONUsername string `query:"uname"`
	RCONPassword string `header:"X-RCON-Pass"`
}

func RCONStream(c echo.Context) error {
	var params RCONStreamParams
	if err := c.Bind(params); err != nil {
		return err
	}
	rcon, err := rcon.Dial(params.RCONUsername, params.RCONPassword)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}
	defer rcon.Close()
	for {
		resp, _, err := rcon.Read()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		strstrm := strings.NewReader(resp)
		c.Response().Header().Add("Transfer Encoding", "chunked")
		c.Stream(200, "text/event-stream", strstrm)
	}
}
