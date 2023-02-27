package models

import (
	"context"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Mongo = InitMongo()
var RDB = InitRedis()

func InitMongo() *mongo.Database {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// 和mongoDB建立连接
	client, err := mongo.Connect(ctx, options.Client().SetAuth(options.Credential{
		Username: "admin",
		Password: "admin",
	}).ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Println("connection mongodb error:", err)
		return nil
	}
	db := client.Database("im") // 连接设置的database
	return db
}

func InitRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})
}
