package main

import (
	"bytes"
	"context"
	"encoding/json"
	//"fmt"
	
	"io/ioutil"
	"net/http"
	"log"
	"time"
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

type symbolsName struct {
	Symbol string `json:"symbol"`
}

type checkResponse struct {
	Result bool `json:"result"`
}

type myError struct {
	Error string `json:"error"`
}

var client *mongo.Client
var collection *mongo.Collection
var collectionSymbols *mongo.Collection

func addSymbol(c *gin.Context) {
	cookie, err := c.Request.Cookie("csrftoken")
	if err != nil { log.Fatal(err); return }

	//check for the admin

	values := map[string]string{"userid": cookie.Value}
	json_data, err := json.Marshal(values)
	if err != nil { c.IndentedJSON(http.StatusNotAcceptable, err); return }
	resp, err := http.Post("http://localhost:8000/admin", "application/json", bytes.NewBuffer(json_data))
	if err != nil { c.IndentedJSON(http.StatusNotAcceptable, err); return }

	var res map[string]interface{}

	json.NewDecoder(resp.Body).Decode(&res)

	if(!res["isadmin"].(bool)) {
		c.IndentedJSON(http.StatusNotAcceptable, myError{"You do not have enough rights .-."})
		return
	}

	//add symbol

	var newSymbol symbolsName
	var checkSymbol symbolsName

	err = c.BindJSON(&newSymbol)
	if err != nil { log.Fatal(err); return; }
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = collectionSymbols.FindOne(ctx, bson.M{"symbol": newSymbol.Symbol}).Decode(&checkSymbol)

	if err == mongo.ErrNoDocuments {

		_, err := collectionSymbols.InsertOne(ctx, newSymbol)
		if(err != nil) { log.Fatal(err); return; }

		c.IndentedJSON(http.StatusCreated, "Symbol created")
		return
	}
	if(err != nil) { log.Fatal(err); return; }
	c.IndentedJSON(http.StatusBadRequest, "Symbol already created")
}

func addAsset(c *gin.Context) {
	cookie, err := c.Request.Cookie("csrftoken")
	if err != nil { log.Fatal(err); return }

	//check for the admin

	values := map[string]string{"userid": cookie.Value}
	json_data, err := json.Marshal(values)
	if err != nil { c.IndentedJSON(http.StatusNotAcceptable, err); return }
	resp, err := http.Post("http://localhost:8000/admin", "application/json", bytes.NewBuffer(json_data))
	if err != nil { c.IndentedJSON(http.StatusNotAcceptable, err); return }

	var res map[string]interface{}

	json.NewDecoder(resp.Body).Decode(&res)

	if(!res["isadmin"].(bool)) {
		c.IndentedJSON(http.StatusNotAcceptable, myError{"You do not have enough rights .-."})
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

		c.IndentedJSON(http.StatusCreated, "Asset created")
		return
	}
	if(err != nil) { log.Fatal(err); return; }
	c.IndentedJSON(http.StatusBadRequest, "Asset already created")
}


func checkSymbol(c *gin.Context) {
	var checkName symbolsName
	var result symbolsName
	var resp checkResponse

	err := c.BindJSON(&checkName)
	if err != nil { log.Fatal(err); return; }
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = collection.FindOne(ctx, bson.M{"symbol": checkName.Symbol}).Decode(&result)
	if err == mongo.ErrNoDocuments { 
		resp.Result = false 
	} else { 
		if err != nil { log.Fatal(err); return; } 
		resp.Result = true
	}
	c.IndentedJSON(http.StatusCreated, resp)
}

func checkFiat(c *gin.Context) {
	var checkName assetName
	var result asset
	var resp checkResponse

	err := c.BindJSON(&checkName)
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
	c.IndentedJSON(http.StatusCreated, resp)
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

	collection = client.Database("assetsSevices").Collection("assets")
	collectionSymbols = client.Database("assetsSevices").Collection("symbols")

	router.POST("/assets/fiat", checkFiat)
	router.POST("/admin/assets", addAsset)
	router.POST("/admin/symbols", addSymbol)
	router.Run("localhost:8001")
}