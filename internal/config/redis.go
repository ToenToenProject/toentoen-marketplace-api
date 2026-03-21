package config

import (
	"context"
	"log"
	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client
var Ctx = context.Background()

func ConnectRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379", 
		Password: "",              
		DB:       0,                
	})

	if err := RDB.Ping(Ctx).Err(); err != nil {
		log.Fatalf("Gagal terhubung ke Redis: %v", err)
	}
	log.Println("Berhasil terhubung ke Redis!")
}