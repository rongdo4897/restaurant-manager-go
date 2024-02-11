package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func DBInstance() *mongo.Client {
	MongoURL := "mongodb://localhost:27017"

	client, err := mongo.NewClient(options.Client().ApplyURI(MongoURL))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("================================================")
	fmt.Println("Connected to MongoDB")
	return client
}

// Tạo 1 biến toàn cục cho Client sau khi kết nối
var Client *mongo.Client = DBInstance()

func OpenConnection(client *mongo.Client, collectionName string) *mongo.Collection {
	var collection = client.Database("restaurant").Collection(collectionName)
	return collection
}
