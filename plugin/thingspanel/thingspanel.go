package thingspanel

import (
	"go.uber.org/zap"

	"github.com/DrmagicE/gmqtt/config"
	"github.com/DrmagicE/gmqtt/server"
)

var _ server.Plugin = (*Thingspanel)(nil)

const Name = "thingspanel"

func init() {
	server.RegisterPlugin(Name, New)
	config.RegisterDefaultPluginConfig(Name, &DefaultConfig)
}

func New(config config.Config) (server.Plugin, error) {
	//panic("implement me")
	return &Thingspanel{}, nil
}

var Log *zap.Logger

type Thingspanel struct {
}

func (t *Thingspanel) Load(service server.Server) error {
	Log = server.LoggerWithField(zap.String("plugin", Name))
	return nil
}

func (t *Thingspanel) Unload() error {
	return nil
}

func (t *Thingspanel) Name() string {
	return Name
}
