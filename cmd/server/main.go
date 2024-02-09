package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	port := flag.Int("port", 8080, "Port")
	flag.Parse()
	addr := fmt.Sprintf(":%d", *port)
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(echoprometheus.NewMiddleware("app"))

	e.GET("/metrics", echoprometheus.NewHandler())
	e.GET("/", homeHandler)

	e.Logger.Fatal(e.Start(addr))
}

func homeHandler(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
