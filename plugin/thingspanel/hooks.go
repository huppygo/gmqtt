package thingspanel

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/DrmagicE/gmqtt/server"
)

func (t *Thingspanel) HookWrapper() server.HookWrapper {
	return server.HookWrapper{
		OnBasicAuthWrapper:  t.OnBasicAuthWrapper,
		OnSubscribeWrapper:  t.OnSubscribeWrapper,
		OnMsgArrivedWrapper: t.OnMsgArrivedWrapper,
	}
}

func (t *Thingspanel) OnBasicAuthWrapper(pre server.OnBasicAuth) server.OnBasicAuth {
	return func(ctx context.Context, client server.Client, req *server.ConnectRequest) (err error) {
		// 处理前一个插件的OnBasicAuth逻辑
		err = pre(ctx, client, req)
		if err != nil {
			return err
		}
		// ... 处理本插件的鉴权逻辑
		Log.Info("鉴权Username：" + string(req.Connect.Username))
		Log.Info("鉴权Password：" + string(req.Connect.Password))
		return nil
	}
}

func (t *Thingspanel) OnSubscribeWrapper(pre server.OnSubscribe) server.OnSubscribe {
	return func(ctx context.Context, client server.Client, req *server.SubscribeRequest) error {
		//root放行
		if client.ClientOptions().Username == "root" {
			return nil
		}
		if client.ClientOptions().Username == "gateway" {
			return nil
		}
		// ... 只允许sub_list中的主题可以被订阅
		the_sub := req.Subscribe.Topics[0].Name
		if find := strings.Contains(the_sub, "custom/sub/"+client.ClientOptions().Username+"/"); find {
			return nil
		}
		flag := false
		var sub_list = [3]string{"device/attributes", "device/event", "device/serves"}
		for _, sub := range sub_list {
			if the_sub == sub+string(client.ClientOptions().Username) {
				flag = true
			}
		}
		if flag {
			return nil
		} else {
			err := errors.New("permission denied;")
			return err
		}
	}
}

func (t *Thingspanel) OnMsgArrivedWrapper(pre server.OnMsgArrived) server.OnMsgArrived {
	return func(ctx context.Context, client server.Client, req *server.MsgArrivedRequest) (err error) {
		// root放行
		if client.ClientOptions().Username == "root" {
			return nil
		}
		//v1/gateway/telemetry
		// ...v1/gateway/telemetry
		if client.ClientOptions().Username == "gateway" {
			type UtilsFunRoot struct {
				Ts     int         `json:"ts"`
				Values interface{} `json:"values"`
			}
			type UtilsFun struct {
				Root []UtilsFunRoot `json:"root"`
			}

			// 消息重写
			m := UtilsFun{}
			json_err := json.Unmarshal(req.Message.Payload, &m)
			if json_err != nil {
				return errors.New("umarshal failed;")
			}
			if string(req.Publish.TopicName) == "v1/gateway/telemetry" {
				mm := make(map[string]interface{})
				mm["token"] = m.Root
				mm["values"] = m.Root[0].Values
				mjson, _ := json.Marshal(mm)
				Log.Info(string(mjson))
				req.Message.Payload = mjson
				return nil
			}
		}
		// ... 只允许sub_list中的主题可以发布
		the_pub := string(req.Publish.TopicName)
		if find := strings.Contains(the_pub, "custom/pub/"+client.ClientOptions().Username+"/"); find {
			return nil
		}
		flag := false
		var pub_list = [3]string{"device/attributes", "device/event", "device/serves"}
		for _, pub := range pub_list {
			if the_pub == pub {
				flag = true
			}
		}
		if !flag {
			err := errors.New("permission denied;")
			return err
		}
		// 校验消息是否是json
		if !json.Valid(req.Message.Payload) {
			err := errors.New("the message is not valid;")
			return err
		}
		// 消息重写
		m := make(map[string]interface{})
		json_err := json.Unmarshal(req.Message.Payload, &m)
		if json_err != nil {
			return errors.New("umarshal failed;")
		}
		m["token"] = client.ClientOptions().Username
		mjson, _ := json.Marshal(m)
		Log.Info(string(mjson))
		req.Message.Payload = mjson
		return nil
	}
}
