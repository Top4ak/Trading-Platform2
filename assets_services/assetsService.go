package main

import (
	//"bytes"
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	//"fmt"
	"io/ioutil"

	//"fmt"
	"net/http"

	//"fmt"
	"log"
	//"reflect"

	//"reflect"
	"time"

	//"net/http"
	//"net/url"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	//"go.mongodb.org/mongo-driver/mongo/readpref"
)

type asset struct {
	_ID    		primitive.ObjectID `json:"_id"`
	Name		string 		`json:"name"`
	Fiat	 	bool 		`json:"fiat"`
}

type assetName struct {
	Name string `json:"name"`
}

type checkFiatResponse struct {
	Result bool `json:"result"`
}

var client *mongo.Client
var collection *mongo.Collection

func addAsset(c *gin.Context) {
	cookie, err := c.Request.Cookie("csrftoken")
	if err != nil { log.Fatal(err) }

	//check for the admin

	values := map[string]string{"userid": cookie.Value}
	json_data, err := json.Marshal(values)
	if err != nil { log.Fatal(err); }
	resp, err := http.Post("http://localhost:8000/admin", "application/json", bytes.NewBuffer(json_data))
	if err != nil { log.Fatal(err); }

	var res map[string]interface{}

	json.NewDecoder(resp.Body).Decode(&res)

	if(!res["isadmin"].(bool)) {
		c.IndentedJSON(http.StatusNotAcceptable, "You do not have enough rights .-.")
		return
	}

	//add asset

	var newAsset asset
	var checkAssets asset

	err = c.BindJSON(&newAsset)
	if err != nil { log.Fatal(err); return; }
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = collection.FindOne(ctx, bson.M{"name": newAsset.Name}).Decode(&checkAssets)

	if err == mongo.ErrNoDocuments {

		_, err := collection.InsertOne(ctx, newAsset)
		if(err != nil) { log.Fatal(err); return; }

		c.IndentedJSON(http.StatusCreated, http.StatusCreated)
		return
	}
	if(err != nil) { log.Fatal(err); return; }
	c.IndentedJSON(http.StatusBadRequest, "Asset already created")
}

func checkFiat(c *gin.Context) {
	var checkName assetName
	var result asset
	var resp checkFiatResponse

	err := c.BindJSON(&checkName)
	fmt.Println(checkName.Name)
	if err != nil { log.Fatal(err); return; }
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = collection.FindOne(ctx, bson.M{"name": checkName.Name}).Decode(&result)
	resp.Result = result.Fiat
	if err == mongo.ErrNoDocuments { 
		resp.Result = false 
	} else { 
		if err != nil { log.Fatal(err); return; } 
	}
	fmt.Println(resp.Result)
	c.IndentedJSON(http.StatusCreated, resp)
}

func main() {
	router := gin.Default()

	content, err := ioutil.ReadFile("../dbConnectorURI.txt")
	if err != nil { log.Fatal(err); }

	client, err := mongo.NewClient(options.Client().ApplyURI(string(content)))	
	if err != nil { log.Fatal(err); }

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil { log.Fatal(err); }
	defer client.Disconnect(ctx)

	collection = client.Database("assetsSevices").Collection("assets")
	//collection = client.Database("newdb").Collection("assets")

	router.POST("/assets/fiat", checkFiat)
	router.POST("/admin/assets", addAsset)
	router.Run("localhost:8001")
}