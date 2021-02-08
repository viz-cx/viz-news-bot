package models

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TelegramChannel struct {
	ID          primitive.ObjectID `bson:"_id"`
	ChannelID   int64              `bson:"channel_id"`
	Title       string             `bson:"title"`
	UserName    string             `bson:"username"`
	Description string             `bson:"description"`
	CreatedAt   time.Time          `bson:"created_at"`
	BlockID     uint32             `bson:"block_id"`
}

const dbName = "viz_news"
const collectionName = "telegram_channels"

func SaveTelegramChannel(channel TelegramChannel) error {
	client, err := GetMongoClient()
	if err != nil {
		return err
	}
	collection := client.Database(dbName).Collection(collectionName)
	_, err = collection.InsertOne(context.TODO(), channel)
	if err != nil {
		return err
	}
	return nil
}

func GetTelegramChannel(channelID int64) (TelegramChannel, error) {
	result := TelegramChannel{}
	filter := bson.D{primitive.E{Key: "channel_id", Value: channelID}}
	client, err := GetMongoClient()
	if err != nil {
		return result, err
	}
	collection := client.Database(dbName).Collection(collectionName)
	err = collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func GetLastBlock() (uint32, error) {
	channel := TelegramChannel{}
	findOptions := options.FindOneOptions{}
	findOptions.SetSort(bson.M{"created_at": -1})
	client, err := GetMongoClient()
	if err != nil {
		return 0, err
	}
	collection := client.Database(dbName).Collection(collectionName)
	err = collection.FindOne(context.TODO(), bson.D{}, &findOptions).Decode(&channel)
	if err != nil {
		return 0, err
	}
	return channel.BlockID, nil
}
