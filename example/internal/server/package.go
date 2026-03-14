package server

import (
	"example/internal/handler"
	"example/internal/repository"
	"example/internal/service"

	"github.com/samber/do/v2"
)

var Package = do.Package(
	repository.Package,
	service.Package,
	handler.Package,
	do.Lazy(NewServer),
)