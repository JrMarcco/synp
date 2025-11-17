package ioc

import (
	"fmt"

	"github.com/JrMarcco/synp/internal/pkg/codec"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var CodecFxOpt = fx.Module("codec", fx.Provide(initCodec))

func initCodec() codec.Codec {
	type config struct {
		Type string `mapstructure:"type"` // "json" or "proto"
	}

	cfg := config{}
	if err := viper.UnmarshalKey("synp.codec", &cfg); err != nil {
		panic(err)
	}

	switch cfg.Type {
	case "json":
		return codec.NewJSONCodec()
	case "proto":
		return codec.NewProtoCodec()
	default:
		panic(fmt.Errorf("unsupported codec type: %s, expected 'json' or 'proto'", cfg.Type))
	}
}
