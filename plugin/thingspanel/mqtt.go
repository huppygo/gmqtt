package thingspanel

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MqttClient struct {
	Client mqtt.Client
	IsFlag bool
}

var DefaultMqttClient *MqttClient = &MqttClient{}

func (c *MqttClient) MqttInit() error {
	time.Sleep(5 * time.Second)
	// 掉线重连
	var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
		fmt.Printf("Mqtt Connect lost: %v", err)
		i := 0
		for {

			time.Sleep(5 * time.Second)
			if !c.Client.IsConnectionOpen() {
				i++
				fmt.Println("Mqtt客户端掉线重连...", i)
				if token := c.Client.Connect(); token.Wait() && token.Error() != nil {
					fmt.Println("Mqtt客户端连接失败...")
				} else {
					break
				}
			} else {
				//subscribe(msgProc1, gatewayMsgProc)
				break
			}
		}
	}
	opts := mqtt.NewClientOptions()
	opts.SetUsername("root")
	opts.SetPassword("root")
	//opts.SetClientID("e66e392a-84dd")
	opts.AddBroker("127.0.0.1:1883")
	opts.SetAutoReconnect(true)
	opts.SetOrderMatters(false)
	opts.OnConnectionLost = connectLostHandler
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		fmt.Println("Mqtt客户端已连接")
	})
	c.Client = mqtt.NewClient(opts)
	reconnec_number := 0
	go func() {
		for { // 失败重连
			if token := c.Client.Connect(); token.Wait() && token.Error() != nil {
				reconnec_number++
				fmt.Println("Mqtt客户端连接失败...重试", reconnec_number)
			} else {
				fmt.Println("Mqtt客户端重连成功")
				break
			}
			time.Sleep(5 * time.Second)
		}
	}()
	// Log.Error("连接MqttClIent...")
	// if token := c.Client.Connect(); token.Wait() && token.Error() != nil {
	// 	Log.Error("MqttClIent连接失败...")
	// }
	return nil
}

func (c *MqttClient) SendData(topic string, data []byte) error {
	go func() {
		if !DefaultMqttClient.Client.IsConnected() {
			i := 1
			for {
				fmt.Println("等待...", i)
				if i == 10 || DefaultMqttClient.Client.IsConnected() {
					break
				}
				time.Sleep(1 * time.Second)
				i++
			}
		}
		if token := c.Client.Publish(topic, 1, true, string(data)); token.Wait() && token.Error() != nil {
			Log.Info("发送设备状态失败")
		}
	}()

	return nil
}
