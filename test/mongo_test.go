package test

import (
	"IM/models"
	"context"
	"fmt"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestFindOne(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// 和mongoDB建立连接
	client, err := mongo.Connect(ctx, options.Client().SetAuth(options.Credential{
		Username: "admin",
		Password: "admin",
	}).ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatal(err)
	}
	db := client.Database("im") // 连接设置的database
	ub := new(models.UserBasic)
	// 将查询到的内容解析到ub中
	err = db.Collection("user_basic").FindOne(context.Background(), bson.D{}).Decode(ub)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("userbasic ====> ", ub)
}

func TestFind(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// 和mongoDB建立连接
	client, err := mongo.Connect(ctx, options.Client().SetAuth(options.Credential{
		Username: "admin",
		Password: "admin",
	}).ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatal(err)
	}
	db := client.Database("im") // 连接设置的database

	// 将查询到的内容解析到ub中
	urs := make([]*models.UserRoom, 0)
	cursor, err := db.Collection("user_room").Find(context.Background(), bson.D{})
	for cursor.Next(context.Background()) {
		ur := new(models.UserRoom)
		err = cursor.Decode(ur)
		if err != nil {
			t.Fatal(err)
		}
		urs = append(urs, ur)
	}
	for _, ur := range urs {
		fmt.Println("userroom ====> ", ur)
	}
}
