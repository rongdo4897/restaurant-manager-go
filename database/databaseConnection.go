package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Trả về 1 client kết nối tới mongo
func DBInstance() *mongo.Client {
	MongoURL := "mongodb://development:testpassword@localhost:27017"
	clientOptions := options.Client().ApplyURI(MongoURL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Tạo kết nối
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Kiểm tra kết nối tới mongo
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal("failed to connect to mongodb: ", err)
		return nil
	}

	fmt.Println("Successfully connected to mongodb")

	return client
}

// Tạo 1 biến toàn cục cho Client sau khi kết nối
var Client *mongo.Client = DBInstance()

// Kết nối tới database `restaurant` và trả về bảng được chỉ định `collectionName`
func OpenCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	var collection = client.Database("restaurant").Collection(collectionName)
	return collection
}
