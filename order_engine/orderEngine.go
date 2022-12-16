package main

import (
	"bytes"
	"context"
	"fmt"

	//"fmt"
	"net/http"

	"encoding/json"
	"io/ioutil"

	"log"
	//"reflect"
	"time"

	//"net/url"

	"github.com/gin-gonic/gin"
	//"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	//"go.mongodb.org/mongo-driver/mongo/readpref"
)

type order struct {
	_id 		primitive.ObjectID 	`json:"_id"`
	UserId		primitive.ObjectID	`json:"userid"`
	Side 		string				`json:"side"`
	Status 		string 				`json:"status"`
	Symbol 		string				`json:"symbol"`
	Type 		string 				`json:"type"`
	Quantity	float64				`json:"quantity"`
	Price 		float64 			`json:"price"`
}

type myError struct {
	Error string `json:"error"`
}

var client *mongo.Client
var collection *mongo.Collection

func createOrder(c *gin.Context) {
	cookie, err := c.Request.Cookie("csrftoken")
	if err != nil { log.Fatal(err); return }

	var newOrder order
	err = c.BindJSON(&newOrder)
	if(err != nil) { log.Fatal(err); return; }

	newOrder.Status = "ACTIVE"
	newOrder.UserId, err = primitive.ObjectIDFromHex(cookie.Value)
	if(err != nil) { log.Fatal(err); return; }

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	values := map[string]string{"userid": cookie.Value}
	json_data, err := json.Marshal(values)
	if err != nil { c.IndentedJSON(http.StatusNotAcceptable, err); return }
	resp, err := http.Post("http://localhost:8000/assets", "application/json", bytes.NewBuffer(json_data))
	if err != nil { c.IndentedJSON(http.StatusNotAcceptable, err); return }

	var res map[string]interface{}

	json.NewDecoder(resp.Body).Decode(&res)

	fmt.Println(res)
	var assetOperation string
	if(newOrder.Side == "BUY") {
		if(newOrder.Quantity * newOrder.Price > res["eur"].(float64)) {
			c.IndentedJSON(http.StatusBadRequest, myError{"Not enough funds"})
			return
		}
		assetOperation = "EUR"
	} else if(newOrder.Side == "SELL") {
		if(newOrder.Quantity > res["eth"].(float64)) {
			c.IndentedJSON(http.StatusBadRequest, myError{"Not enough funds"})
			return
		}
		assetOperation = "ETH"
	} else {
		c.IndentedJSON(http.StatusNotAcceptable, myError{"Side not correct"})
		return
	}

	newvalues := map[string]string{"symbol": newOrder.Symbol}
	json_data, err = json.Marshal(newvalues)
	if err != nil { c.IndentedJSON(http.StatusNotAcceptable, err); return }
	resp, err = http.Post("http://localhost:8001/check/symbols", "application/json", bytes.NewBuffer(json_data))
	if err != nil { c.IndentedJSON(http.StatusNotAcceptable, err); return }
	json.NewDecoder(resp.Body).Decode(&res)

	if(!res["result"].(bool)) {
		c.IndentedJSON(http.StatusBadRequest, myError{"This couple does not exist."})
		return;
	} 

	newValues := map[string]interface{}{"userid": cookie.Value, "asset": assetOperation, "amount": newOrder.Quantity}
	json_data, err = json.Marshal(newValues)
	if err != nil { c.IndentedJSON(http.StatusNotAcceptable, err); return }
	resp, err = http.Post("http://localhost:8000/assets/change", "application/json", bytes.NewBuffer(json_data))
	if err != nil { c.IndentedJSON(http.StatusNotAcceptable, err); return }

	fmt.Println(assetOperation)
	_, err = collection.InsertOne(ctx, newOrder)
	if(err != nil) { log.Fatal(err); return; }

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

	collection = client.Database("ordersEngine").Collection("orders")
	//collection = client.Database("newdb").Collection("assets")

	router.POST("/order/create", createOrder)
	router.Run("localhost:8003")
}