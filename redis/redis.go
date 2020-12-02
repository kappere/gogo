package redis

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-redis/redis/v8"
	"wataru.com/gogo/logger"
	"wataru.com/gogo/util"
)

var ctx = context.Background()
var Rdb *redis.Client

func NewClient(redisConf map[interface{}]interface{}) {
	host := util.ValueOrDefault(redisConf["host"], "localhost").(string)
	port := util.ValueOrDefault(redisConf["port"], "3306").(int)
	password := util.ValueOrDefault(redisConf["password"], "").(string)
	addr := host + ":" + strconv.Itoa(port)
	Rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // no password set
		DB:       0,        // use default DB
	})
	logger.Info("Initialize redis")

	_, err := Rdb.Ping(ctx).Result()
	if err != nil {
		panic(err.Error())
	}
}

func Test() {
	err := Rdb.Set(ctx, "key", "value", 0).Err()
	if err != nil {
		panic(err)
	}

	val, err := Rdb.Get(ctx, "key").Result()
	if err != nil {
		panic(err)
	}
	fmt.Println("key", val)

	val2, err := Rdb.Get(ctx, "key2").Result()
	if err == redis.Nil {
		fmt.Println("key2 does not exist")
	} else if err != nil {
		panic(err)
	} else {
		fmt.Println("key2", val2)
	}
	// Output: key value
	// key2 does not exist
}
