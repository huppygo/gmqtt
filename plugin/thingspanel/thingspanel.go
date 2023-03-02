package thingspanel

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/DrmagicE/gmqtt/config"
	"github.com/DrmagicE/gmqtt/server"
)

var _ server.Plugin = (*Thingspanel)(nil)

const Name = "thingspanel"

func init() {
	DefaultMqttClient.MqttInit()
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

func (t *Thingspanel) UpdateStatus(accessToken string, status string) {
	url := "/api/device/status"
	method := "POST"
	payload := strings.NewReader(`"accessToken": "` + accessToken + `","values":{"status": "` + status + `"}}`)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}
