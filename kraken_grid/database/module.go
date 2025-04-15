package database

import (
	"context"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
)

const ModuleName = "database"

type Module struct{}

func (m *Module) Configure(app *cli.App) {
	app.Flags = append(app.Flags,
		&cli.StringFlag{
			Name:    "db_driver",
			EnvVars: []string{"DB_DRIVER"},
			Value:   "postgres",
		},
		&cli.StringFlag{
			Name:    "db_dsn",
			EnvVars: []string{"DB_DSN"},
			Value:   "host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable",
		},
	)
}

func (m *Module) Install(ctx *cli.Context) fx.Option {
	return fx.Module(ModuleName,
		fx.Supply(
			fx.Annotate(WithDriver(ctx.String("db_driver")), fx.ResultTags(`group:"options"`)),
			fx.Annotate(WithDSN(ctx.String("db_dsn")), fx.ResultTags(`group:"options"`)),
		),
		fx.Provide(fx.Annotate(NewConfig, fx.ParamTags(`group:"options"`))),
		fx.Provide(
			fx.Annotate(
				NewDB,
				fx.As(new(Database)),
				fx.OnStart(func(ctx context.Context, db Database) error {
					return db.Connect()
				}),
				fx.OnStop(func(ctx context.Context, db Database) error {
					return db.Close()
				}),
			),
		),
	)
}
