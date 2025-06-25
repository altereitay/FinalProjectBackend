package db

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var FileCollection *mongo.Collection

type Article struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	Title      string             `bson:"title"`
	Original   string             `bson:"original"`
	Simplified string             `bson:"simplified,omitempty"`
	Terms      []SingleTerm       `bson:"terms,omitempty"`
	Hash       string             `bson:"hash"`
}

type SingleTerm struct {
	Term       string `bson:"term"`
	Definition string `bson:"definition"`
}

func InitMongo() error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoUri := os.Getenv("MONGO_URI")
	
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoUri))
	if err != nil {
		log.Fatal("Mongo connect error:", err)
		return err
	}

	FileCollection = client.Database("articles").Collection("articles")
	return nil
}

func InsertNewArticle(doc Article) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	_, err := FileCollection.InsertOne(ctx, doc)
	return err
}

func AddSimplifiedVersion (id string, simple string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)

	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set":bson.M{
			"simplified": simple,
		}
	}

	_, err = FileCollection.UpdateByID(ctx, objID, update)
	return err
}

func AddTerms (id string, terms []SingleTerm) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)

	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set":bson.M{
			"terms": terms,
		}
	}

	_, err = FileCollection.UpdateByID(ctx, objID, update)
	return err
}
