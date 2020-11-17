package api

import (
	"github.com/labstack/echo/v4"

	"github.com/trezcool/masomo/backend/api/handlers"
)

type server struct {
	addr   string
	router *echo.Echo
}

func NewServer(addr string) *server {
	s := &server{
		addr:   addr,
		router: echo.New(),
	}
	handlers.API(s.router)
	return s
}

func (s *server) Start() {
	s.router.Logger.Fatal(s.router.Start(s.addr))
}
