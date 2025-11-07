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

	upgrader    synp.Upgrader
	connManager synp.ConnManager
	connHandler synp.Handler

	Logger    *zap.Logger
	Lifecycle fx.Lifecycle
}

func InitApp(params appFxParams) *App {
	wsCfg := ws.DefaultConfig()
	wsSvr := ws.NewServer(
		wsCfg, params.upgrader, params.connManager, params.connHandler, params.Logger,
	)

	app := &App{
		wsSvr: wsSvr,
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return app.Start()
		},
		OnStop: func(ctx context.Context) error {
			return app.Stop()
		},
	})

	return app
}
