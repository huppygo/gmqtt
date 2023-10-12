package thingspanel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/DrmagicE/gmqtt/server"
)

func (t *Thingspanel) HookWrapper() server.HookWrapper {
	return server.HookWrapper{
		OnBasicAuthWrapper:  t.OnBasicAuthWrapper,
		OnSubscribeWrapper:  t.OnSubscribeWrapper,
		OnMsgArrivedWrapper: t.OnMsgArrivedWrapper,
		OnConnectedWrapper:  t.OnConnectedWrapper,
		OnClosedWrapper:     t.OnClosedWrapper,
	}
}

func (t *Thingspanel) OnBasicAuthWrapper(pre server.OnBasicAuth) server.OnBasicAuth {
	return func(ctx context.Context, client server.Client, req *server.ConnectRequest) (err error) {
		// 处理前一个插件的OnBasicAuth逻辑
		err = pre(ctx, client, req)
		if err != nil {
			Log.Error(err.Error())
			return err
		}
		// ... 处理本插件的鉴权逻辑
		Log.Info("鉴权Username：" + string(req.Connect.Username))
		Log.Info("鉴权Password：" + string(req.Connect.Password))
		return nil
	}
}

func (t *Thingspanel) OnConnectedWrapper(pre server.OnConnected) server.OnConnected {
	return func(ctx context.Context, client server.Client) {
		// 客户端连接后
		// 主题：device/status
		// 报文：{"token":username,"SYS_STATUS":"online"}
		// username为客户端用户名
		Log.Info("----------------------------------------")

		if client.ClientOptions().Username != "root" {
			jsonData := fmt.Sprintf(`{"accessToken":"%s","values":{"status":"1"}}`, client.ClientOptions().Username)
			if err := DefaultMqttClient.SendData("device/status", []byte(jsonData)); err != nil {
				Log.Warn("上报状态失败")
			}
		}
	}
}
func (t *Thingspanel) OnClosedWrapper(pre server.OnClosed) server.OnClosed {
	return func(ctx context.Context, client server.Client, err error) {
		// 客户端断开连接后
		// 主题：device/status
		// 报文：{"token":username,"SYS_STATUS":"offline"}
		// username为客户端用户名
		Log.Info("----------------------------------------")
		if client.ClientOptions().Username != "root" {
			jsonData := fmt.Sprintf(`{"accessToken":"%s","values":{"status":"0"}}`, client.ClientOptions().Username)
			if err := DefaultMqttClient.SendData("device/status", []byte(jsonData)); err != nil {
				Log.Warn("上报状态失败")
			}
		}
	}
}

// 订阅消息钩子函数
func (t *Thingspanel) OnSubscribeWrapper(pre server.OnSubscribe) server.OnSubscribe {
	return func(ctx context.Context, client server.Client, req *server.SubscribeRequest) error {
		username := client.ClientOptions().Username
		//root放行
		if username == "root" {
			return nil
		}
		if username == "gateway" {
			return nil
		}
		// ... 只允许sub_list中的主题可以被订阅
		the_sub := req.Subscribe.Topics[0].Name
		if err := OtherOnSubscribeWrapper(the_sub, username); err == nil {
			return nil
		}
		if find := strings.Contains(the_sub, "custom/sub/"+username+"/"); find {
			return nil
		}
		flag := false
		var sub_list = [8]string{"device/attributes/", "device/event/", "device/command/", "gateway/attributes/", "gateway/event/", "gateway/command/", "attributes/relaying/", "ota/device/inform/"}
		for _, sub := range sub_list {
			if the_sub == sub+string(username) {
				flag = true
			}
		}
		if flag {
			return nil
		} else {
			err := errors.New("permission denied,Please subscribe to the allowed topics in accordance with the official connection specifications;")
			return err
		}
	}
}

func (t *Thingspanel) OnMsgArrivedWrapper(pre server.OnMsgArrived) server.OnMsgArrived {
	return func(ctx context.Context, client server.Client, req *server.MsgArrivedRequest) (err error) {
		username := client.ClientOptions().Username
		// root用户放行
		if username == "root" {
			RootMessageForwardWrapper(req.Message.Topic, req.Message.Payload, false)
			return nil
		}
		// gateway用户，消息重新
		if username == "gateway" {
			// 消息解析
			type UtilsFunRoot struct {
				Ts     int         `json:"ts"`
				Values interface{} `json:"values"`
			}
			// 消息重写
			msgMap := make(map[string][]UtilsFunRoot)
			json_err := json.Unmarshal(req.Message.Payload, &msgMap)
			if json_err != nil {
				return errors.New("umarshal failed;")
			}
			if string(req.Publish.TopicName) == "v1/gateway/telemetry" {
				newMsgMap := make(map[string]interface{})
				for key := range msgMap {
					newMsgMap["token"] = key
					newMsgMap["values"] = msgMap[key][0].Values
				}
				mjson, _ := json.Marshal(newMsgMap)
				req.Message.Payload = mjson
				return nil
			}
			return nil
		}
		// ... 只允许sub_list中的主题可以发布
		the_pub := string(req.Publish.TopicName)
		// 额外的主题转换
		if t, err := OtherOnMsgArrivedWrapper(the_pub, username); err == nil {
			the_pub = t
		}
		if find := strings.Contains(the_pub, "custom/pub/"+username+"/"); find {
			return nil
		}
		flag := false
		var pub_list = [8]string{
			"device/attributes",   //属性上报
			"device/event",        //事件上报
			"device/command",      //命令下发
			"gateway/attributes",  //网关属性上报
			"gateway/event",       //网关事件上报
			"gateway/command",     //网关命令调用
			"ota/device/inform",   //设备升级通知
			"ota/device/progress", //设备升级进度
		}
		for _, pub := range pub_list {
			if the_pub == pub {
				flag = true
			}
		}
		if !flag {
			err := errors.New("permission denied;")
			return err
		}
		//属性上报消息转发
		if the_pub == "device/attributes" || the_pub == "gateway/attributes" {
			if err := DefaultMqttClient.SendData("attributes/relaying/"+username, req.Message.Payload); err != nil {
				Log.Info("消息转发失败")
			}
		}
		// 消息重写
		newMsgMap := make(map[string]interface{})
		newMsgMap["token"] = username
		newMsgMap["values"] = req.Message.Payload
		newMsgJson, _ := json.Marshal(newMsgMap)
		req.Message.Payload = newMsgJson
		// 如果原主题被转换，丢弃消息，重新发布到转换后的主题
		if the_pub != string(req.Publish.TopicName) {
			DefaultMqttClient.SendData(the_pub, req.Message.Payload)
			return errors.New("the topic is converted, the message is discarded;")
		}
		return nil
	}
}
