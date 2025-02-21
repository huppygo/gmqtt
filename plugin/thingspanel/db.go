package thingspanel

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/redis.v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var redisCache *redis.Client
var db *gorm.DB

type Device struct {
	ID             string `json:"id" gorm:"primaryKey,size:36"`
	AssetID        string `json:"asset_id,omitempty" gorm:"size:36"`              // 资产id
	Token          string `json:"token,omitempty"`                                // 安全key
	AdditionalInfo string `json:"additional_info,omitempty" gorm:"type:longtext"` // 存储基本配置
	CustomerID     string `json:"customer_id,omitempty" gorm:"size:36"`
	Type           string `json:"type,omitempty"` // 插件类型
	Name           string `json:"name,omitempty"` // 插件名
	Label          string `json:"label,omitempty"`
	SearchText     string `json:"search_text,omitempty"`
	ChartOption    string `json:"chart_option,omitempty"  gorm:"type:longtext"` // 插件( 目录名)
	Protocol       string `json:"protocol,omitempty" gorm:"size:50"`
	Port           string `json:"port,omitempty" gorm:"size:50"`
	Publish        string `json:"publish,omitempty" gorm:"size:255"`
	Subscribe      string `json:"subscribe,omitempty" gorm:"size:255"`
	Username       string `json:"username,omitempty" gorm:"size:255"`
	Password       string `json:"password,omitempty" gorm:"size:255"`
	DId            string `json:"d_id,omitempty" gorm:"size:255"`
	Location       string `json:"location,omitempty" gorm:"size:255"`
	DeviceType     string `json:"device_type,omitempty" gorm:"size:2"`
	ParentId       string `json:"parent_id,omitempty" gorm:"size:36"`
	ProtocolConfig string `json:"protocol_config,omitempty" gorm:"type:longtext"`
	SubDeviceAddr  string `json:"sub_device_addr,omitempty" gorm:"size:36"`
	ScriptId       string `json:"script_id,omitempty" gorm:"size:36"`
	CreatedAt      int64  `json:"created_at,omitempty"`
	ProductId      string `json:"product_id,omitempty" gorm:"size:36"`
	CurrentVersion string `json:"current_version,omitempty" gorm:"size:36"`
	TenantId       string `json:"tenant_id,omitempty" gorm:"size:36"` // 租户id
}

func (Device) TableName() string {
	return "device"
}

// 创建 redis 客户端
func createRedisClient() *redis.Client {
	redisHost := viper.GetString("db.redis.conn")
	dataBase := viper.GetInt("db.redis.db_num")
	password := viper.GetString("db.redis.password")
	log.Println("连接redis...")
	client := redis.NewClient(&redis.Options{
		Addr:         redisHost,
		Password:     password,
		DB:           dataBase,
		ReadTimeout:  2 * time.Minute,
		WriteTimeout: 1 * time.Minute,
		PoolTimeout:  2 * time.Minute,
		IdleTimeout:  10 * time.Minute,
		PoolSize:     1000,
	})

	// 通过 cient.Ping() 来检查是否成功连接到了 redis 服务器
	_, err := client.Ping().Result()
	if err != nil {
		log.Println("连接redis连接失败,", err)
		panic(err)
	} else {
		log.Println("连接redis成完成...")
	}

	return client
}

func createPgClient() *gorm.DB {
	psqladdr := viper.GetString("db.psql.psqladdr")
	psqlport := viper.GetInt("db.psql.psqlport")
	psqluser := viper.GetString("db.psql.psqluser")
	psqlpass := viper.GetString("db.psql.psqlpass")
	psqldb := viper.GetString("db.psql.psqldb")
	connectionString := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=disable", psqluser, psqlpass, psqldb, psqladdr, psqlport)
	// 连接数据库
	log.Println("连接数据库...")
	d, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{})

	if err != nil {
		panic(err)
	} else {
		log.Println("连接数据库成功...")
	}
	return d
}

func Init() {
	redisCache = createRedisClient()
	db = createPgClient()
}

func SetStr(key, value string, time time.Duration) (err error) {
	err = redisCache.Set(key, value, time).Err()
	if err != nil {
		return err
	}
	return err
}

func GetStr(key string) (value string) {
	v, _ := redisCache.Get(key).Result()
	return v
}

func DelKey(key string) (err error) {
	err = redisCache.Del(key).Err()
	return err
}

// SetNX 尝试获取锁
func SetNX(key, value string, expiration time.Duration) (ok bool, err error) {
	ok, err = redisCache.SetNX(key, value, expiration).Result()
	return
}

// SetNX 释放锁
func DelNX(key string) (err error) {
	err = redisCache.Del(key).Err()
	return
}

// setRedis 将任何类型的对象序列化为 JSON 并存储在 Redis 中
func SetRedisForJsondata(key string, value interface{}, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return redisCache.Set(key, jsonData, expiration).Err()
}

// getRedis 从 Redis 中获取 JSON 并反序列化到指定对象
func GetRedisForJsondata(key string, dest interface{}) error {
	val, err := redisCache.Get(key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

// 通过token从redis中获取设备信息
// 先从redis中获取设备id，如果没有则从数据库中获取设备信息，并将设备信息和token存入redis
func GetDeviceByToken(token string) (*Device, error) {
	var device Device
	deviceId := GetStr(token)
	if deviceId == "" {
		result := db.Model(&Device{}).Where("token = ?", token).First(&device)
		if result.Error != nil {
			Log.Info(result.Error.Error())
			return nil, result.Error
		}
		// 修改token的时候，需要删除旧的token
		// 将token存入redis
		err := SetStr(token, device.ID, 0)
		if err != nil {
			return nil, err
		}
		// 将设备信息存入redis
		err = SetRedisForJsondata(deviceId, device, 0)
		if err != nil {
			return nil, err
		}
	} else {
		d, err := GetDeviceById(deviceId)
		if err != nil {
			return nil, err
		}
		device = *d
	}

	return &device, nil
}

// 通过设备id从redis中获取设备信息
// 先从redis中获取设备信息，如果没有则从数据库中获取设备信息，并将设备信息存入redis
func GetDeviceById(deviceId string) (*Device, error) {
	var device Device
	err := GetRedisForJsondata(deviceId, &device)
	if err != nil {
		result := db.Model(&Device{}).Where("id = ?", deviceId).First(&device)
		if result.Error != nil {
			return nil, result.Error
		}
		// 将设备信息存入redis
		err = SetRedisForJsondata(deviceId, device, 0)
		if err != nil {
			return nil, err
		}
	}
	return &device, nil
}

// 根据token获取订阅信息
type UserPub struct {
	Attribute string `json:"attribute"`
	Event     string `json:"event"`
}
type UserSub struct {
	Attribute string `json:"attribute"`
	Commands  string `json:"commands"`
}
type UserTopic struct {
	UserPub UserPub `json:"user_pub"`
	UserSub UserSub `json:"user_sub"`
}

func GetUserTopicByToken(token string) (*UserTopic, error) {
	var userTopic UserTopic
	device, err := GetDeviceByToken(token)
	if err != nil {
		return nil, err
	}
	if device.AdditionalInfo == "" {
		return nil, fmt.Errorf("empty")
	}
	// 转map
	var additionalInfo map[string]interface{}
	err = json.Unmarshal([]byte(device.AdditionalInfo), &additionalInfo)
	if err != nil {
		return nil, err
	}
	// 判断有没有pub_topic
	if _, ok := additionalInfo["user_topic"]; !ok {
		return nil, fmt.Errorf("empty")
	}
	// additionalInfo["user_topic"]转UserTopic
	userTopicJson, err := json.Marshal(additionalInfo["user_topic"])
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(userTopicJson, &userTopic)
	if err != nil {
		return nil, err
	}
	return &userTopic, nil
}
