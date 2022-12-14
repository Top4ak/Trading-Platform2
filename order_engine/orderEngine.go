package main

import (
	//"bytes"
	"context"
	//"fmt"
	//"net/http"

	//"encoding/json"
	"io/ioutil"

	"log"
	//"reflect"
	"time"

	//"net/url"

	"github.com/gin-gonic/gin"
	//"go.mongodb.org/mongo-driver/bson"
	//"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	//"go.mongodb.org/mongo-driver/mongo/readpref"
)

var client *mongo.Client
var collection *mongo.Collection

func main() {
	router := gin.Default()

	content, err := ioutil.ReadFile("../dbConnectorURI.txt")
	if err != nil { log.Fatal(err) }

	client, err := mongo.NewClient(options.Client().ApplyURI(string(content)))	
	if err != nil { log.Fatal(err) }

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil { log.Fatal(err) }
	defer client.Disconnect(ctx)

	collection = client.Database("ordersEngine").Collection("orders")
	//collection = client.Database("newdb").Collection("assets")

	
	router.Run("localhost:8003")
}