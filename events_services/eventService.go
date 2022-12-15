package main

import (
	//"bytes"
	"context"
	//"fmt"
	"net/http"

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

type depositEvent struct {
	Event 		string 		`json:"event"`
	UserId    	string		`json:"userid"`
	Asset 		string 		`json:"asset"`
	Amount		float64		`json:"amount"`
}

func deposit(c *gin.Context) {
	var dep depositEvent
	err := c.BindJSON(&dep)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = collection.InsertOne(ctx, dep)
	if err != nil {log.Fatal(err) }
	c.IndentedJSON(http.StatusOK, "")
}

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

	collection = client.Database("eventsServices").Collection("events")
	//collection = client.Database("newdb").Collection("assets")

	router.POST("/event/deposit", deposit)
	router.Run("localhost:8002")
}