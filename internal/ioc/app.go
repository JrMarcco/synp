package ioc

import (
	"context"

	"github.com/JrMarcco/synp"
	"github.com/JrMarcco/synp/internal/ws"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var AppFxOpt = fx.Module("app", fx.Provide(InitApp))

type App struct {
	wsSvr synp.Server
}

func (app *App) Start() error {
	return app.wsSvr.Start()
}

func (app *App) Stop() error {
	return app.wsSvr.Shutdown()
}

type appFxParams struct {
	fx.In

	Upgrader    synp.Upgrader
	ConnManager synp.ConnManager
	ConnHandler synp.Handler

	Logger    *zap.Logger
	Lifecycle fx.Lifecycle
}

func InitApp(params appFxParams) *App {
	wsCfg := ws.DefaultConfig()
	wsSvr := ws.NewServer(
		wsCfg, params.Upgrader, params.ConnManager, params.ConnHandler, params.Logger,
	)

	app := &App{
		wsSvr: wsSvr,
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			return app.Start()
		},
		OnStop: func(_ context.Context) error {
			return app.Stop()
		},
	})

	return app
}
