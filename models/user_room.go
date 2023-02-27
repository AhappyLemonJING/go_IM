package models

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

type UserRoom struct {
	UserIdentity string `bson:"user_identity"`
	RoomIdentity string `bson:"room_identity"`
	RoomType     int    `bson:"room_type"` // 1私聊房间 2群聊房间
	CreatedAt    int64  `bson:"created_at"`
	UpdatedAt    int64  `bson:"updated_at"`
}

func (UserRoom) CollectionName() string {
	return "user_room"
}

func GetUserRoomByUserIdentityRoomIdentity(userIdentity, roomIdentity string) (*UserRoom, error) {
	ur := new(UserRoom)
	err := Mongo.Collection(UserRoom{}.CollectionName()).
		FindOne(context.Background(), bson.D{{"user_identity", userIdentity}, {"room_identity", roomIdentity}}).Decode(ur)
	return ur, err
}

func GetUserRoomByRoomIdentity(roomIdentity string) ([]*UserRoom, error) {
	// 将查询到的内容解析到ur中
	urs := make([]*UserRoom, 0)
	cursor, err := Mongo.Collection(UserRoom{}.CollectionName()).Find(context.Background(), bson.D{{"room_identity", roomIdentity}})
	if err != nil {
		return nil, err
	}
	for cursor.Next(context.Background()) {
		ur := new(UserRoom)
		err = cursor.Decode(ur)
		if err != nil {
			return nil, err
		}
		urs = append(urs, ur)
	}
	return urs, nil
}

func JudgeUserIsFriend(userIdentity1, userIdentity2 string) bool {
	cursor, err := Mongo.Collection(UserRoom{}.CollectionName()).Find(context.Background(), bson.D{{"user_identity", userIdentity1}, {"room_type", 1}})
	if err != nil {
		return false
	}
	roomIdentity := make([]string, 0)
	for cursor.Next(context.Background()) {
		ur := new(UserRoom)
		err = cursor.Decode(ur)
		if err != nil {
			return false
		}
		roomIdentity = append(roomIdentity, ur.RoomIdentity)
	}
	cnt, err := Mongo.Collection(UserRoom{}.CollectionName()).CountDocuments(context.Background(), bson.M{"user_identity": userIdentity2, "room_type": 1, "room_identity": bson.M{"$in": roomIdentity}})
	if err != nil {
		return false
	}
	if cnt > 0 {
		return true
	}

	return false
}

func GetUserRoomIdentity(userIdentity1, userIdentity2 string) string {
	cursor, err := Mongo.Collection(UserRoom{}.CollectionName()).Find(context.Background(), bson.D{{"user_identity", userIdentity1}, {"room_type", 1}})
	if err != nil {
		return ""
	}
	roomIdentity := make([]string, 0)
	for cursor.Next(context.Background()) {
		ur := new(UserRoom)
		err = cursor.Decode(ur)
		if err != nil {
			return ""
		}
		roomIdentity = append(roomIdentity, ur.RoomIdentity)
	}
	ur2 := new(UserRoom)
	err = Mongo.Collection(UserRoom{}.CollectionName()).FindOne(context.Background(), bson.M{"user_identity": userIdentity2, "room_type": 1, "room_identity": bson.M{"$in": roomIdentity}}).Decode(ur2)
	if err != nil {
		return ""
	}

	return ur2.RoomIdentity
}

func InsertOneUserRoom(ur *UserRoom) error {
	_, err := Mongo.Collection(UserRoom{}.CollectionName()).
		InsertOne(context.Background(), ur)
	return err
}
func DeleteUserRoomByRoomIdentity(roomIdentity string) error {
	_, err := Mongo.Collection(UserRoom{}.CollectionName()).DeleteMany(context.Background(), bson.M{"room_identity": roomIdentity})
	return err
}
