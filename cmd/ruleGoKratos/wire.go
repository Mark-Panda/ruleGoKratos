//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"ruleGoKratos/internal/biz"
	"ruleGoKratos/internal/conf"
	"ruleGoKratos/internal/data"
	"ruleGoKratos/internal/server"
	"ruleGoKratos/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// wireApp init kratos application.
func wireApp(*conf.Bootstrap, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(
		// 提供从Bootstrap中提取的Server和Data
		wire.FieldsOf(new(*conf.Bootstrap), "Server", "Data"),
		server.ProviderSet,
		data.ProviderSet,
		data.PlaygroundProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		newApp,
	))
}
