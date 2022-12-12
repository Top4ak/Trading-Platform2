package main

import (
	//"bytes"
	"context"
	"io/ioutil"
	//"fmt"
	//"net/http"

	//"fmt"
	"log"
	//"reflect"

	//"reflect"
	"time"

	//"net/http"
	//"net/url"

	"github.com/gin-gonic/gin"
	//"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	//"go.mongodb.org/mongo-driver/mongo/readpref"
)

type asset struct {
	_ID    		primitive.ObjectID `json:"_id"`
	Login		string 		`json:"login"`
	Password 	string 		`json:"password"`
	IsAdmin 	bool 		`json:"isadmin"`
	Assets 		[]float64 	`json:"assets"`
}

var client *mongo.Client
var collection *mongo.Collection

func errorCheck(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func addAsset(c *gin.Context) {
	//resp, err := http.Get("https://www.youtube.com/watch?v=yU9kCkCRtzk&ab_channel=SlavaKPSS-Topic/")

	//fmt.Println(resp)
	//c.IndentedJSON(http.StatusOK, resp)

	//jsonBody := []byte(`{"client_message": "hello, server!"}`)
	//bodyReader := bytes.NewReader(jsonBody)
	//req, err := http.NewRequest(http.MethodPost, requestURL, bodyReader)
}

func main() {
	router := gin.Default()

	content, err := ioutil.ReadFile("../dbConnectorURI.txt")
	if err != nil {
        log.Fatal(err)
    }
	client, err := mongo.NewClient(options.Client().ApplyURI(string(content)))	
	errorCheck(err)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	errorCheck(err)
	defer client.Disconnect(ctx)

	collection = client.Database("assetsSevices").Collection("assets")

	router.GET("/admin/assets", addAsset)
	router.Run("localhost:8001")
}