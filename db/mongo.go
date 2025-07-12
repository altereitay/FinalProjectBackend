package db

import (
	"context"
	"errors"
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
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Title      string             `json:"title" bson:"title"`
	Original   string             `json:"original" bson:"original"`
	Simplified string             `json:"simplified" bson:"simplified,omitempty"`
	Terms      []SingleTerm       `json:"terms" bson:"terms,omitempty"`
	Hash       string             `json:"hash" bson:"hash"`
}

type SingleTerm struct {
	Term       string `json:"term" bson:"term"`
	Definition string `json:"definition" bson:"definition"`
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
	log.Println("inserting a new Article")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	_, err := FileCollection.InsertOne(ctx, doc)
	return err
}

func AddSimplifiedVersion(hash string, simple string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	filter := bson.D{{Key: "hash", Value: hash}}

	update := bson.M{
		"$set": bson.M{
			"simplified": simple,
		},
	}

	_, err := FileCollection.UpdateOne(ctx, filter, update)
	return err
}

func AddTerms(hash string, terms []SingleTerm) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	filter := bson.D{{Key: "hash", Value: hash}}

	update := bson.M{
		"$set": bson.M{
			"terms": terms,
		},
	}

	_, err := FileCollection.UpdateOne(ctx, filter, update)
	return err
}

func CheckIfExists(hash string) bool {
	var art Article
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	filter := bson.D{{Key: "hash", Value: hash}}

	err := FileCollection.FindOne(ctx, filter).Decode(&art)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false
	} else {
		return true
	}
}

func GetArticles() ([]Article, error) {
	log.Println("fetching all Articles")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := FileCollection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var articles []Article
	if err := cursor.All(ctx, &articles); err != nil {
		return nil, err
	}

	return articles, nil
}
