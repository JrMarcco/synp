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
		Private string `mapstructure:"private"`
		Public  string `mapstructure:"public"`
	}

	cfg := config{}
	if err := viper.UnmarshalKey("jwt", &cfg); err != nil {
		panic(err)
	}

	jwtManager, err := xjwt.NewEd25519ManagerBuilder[session.UserInfo](cfg.Private, cfg.Public).Build()
	if err != nil {
		panic(err)
	}

	return auth.NewJwtValidator(jwtManager)
}
