package ioc

import (
	"github.com/JrMarcco/jit/xjwt"
	"github.com/JrMarcco/synp/internal/pkg/auth"
	"github.com/JrMarcco/synp/internal/pkg/session"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var ValidatorFxOpt = fx.Module("validator", fx.Provide(InitValidator))

func InitValidator() auth.Validator {
	type config struct {
		Issuer  string `mapstructure:"issuer"`
		Private string `mapstructure:"private"`
		Public  string `mapstructure:"public"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("jwt", &cfg); err != nil {
		panic(err)
	}

	claimsCfg := xjwt.NewClaimsConfig(xjwt.WithIssuer(cfg.Issuer))

	jwtManager, err := xjwt.NewEd25519ManagerBuilder[session.User](cfg.Private, cfg.Public).
		ClaimsConfig(claimsCfg).
		Build()
	if err != nil {
		panic(err)
	}

	return auth.NewJwtValidator(jwtManager)
}
