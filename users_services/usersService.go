package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	//"reflect"

	"time"

	"net/http"
	//"net/url"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type cookie struct {
	UserId string `json:"userId"`
}

type user struct {
	_ID      primitive.ObjectID `json:"_id"`
	Login    string             `json:"login"`
	Password string             `json:"password"`
	IsAdmin  bool               `json:"isadmin"`
	Assets   []float64          `json:"assets"`
}

type depositRequest struct {
	Asset  string  `json:"asset"`
	Amount float64 `json:"amount"`
}

type eventDepositRequest struct {
	Event 		string 	`json:"event"`
	UserId    	string	`json:"userid"`
	Asset 		string 	`json:"asset"`
	Amount		float64	`json:"amount"`
}

type registerResponse struct {
	UserId interface{} `json:"userid"`
}

type userIdResponse struct {
	UserId string `json:"userid"`
}

type depositResponse struct {
	UserId 	string `json:"userid"`
	EUR 	float64 `json:"eur"`
	ETH		float64 `json:"eth"`
}

type myError struct {
	Error string `json:"error"`
}

var albums = []user{
	{Login: "Mike", Password: "qwerty"},
	{Login: "Mike", Password: "qwerty"},
}

var client *mongo.Client
var collection *mongo.Collection

func errorCheck(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cur, err := collection.Find(ctx, bson.D{})
	errorCheck(err)

	var results []user
	if err = cur.All(context.TODO(), &results); err != nil {
		log.Fatal(err)
	}
	c.IndentedJSON(http.StatusOK, results)
}

func createUser(c *gin.Context) {
	var newUser user
	err := c.BindJSON(&newUser)
	errorCheck(err)
	if len([]rune(newUser.Login)) < 4 || len([]rune(newUser.Password)) < 4 {
		c.IndentedJSON(http.StatusBadRequest, myError{"Username or password must contain at least 4 characters"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var checkDB user
	err = collection.FindOne(ctx, bson.M{"login": newUser.Login}).Decode(&checkDB)
	newUser.Assets = append(newUser.Assets, 100)		//registration bonus
	newUser.Assets = append(newUser.Assets, 0.02)
	newUser.IsAdmin = false
	if err == mongo.ErrNoDocuments {

		res, err := collection.InsertOne(ctx, newUser)
		errorCheck(err)

		c.IndentedJSON(http.StatusCreated, registerResponse{res.InsertedID})
		return
	}
	if err == nil {
		c.IndentedJSON(http.StatusBadRequest, myError{"Username is not available"})
	}
	errorCheck(err)
}

func loginUser(c *gin.Context) {
	var logUser user
	err := c.BindJSON(&logUser)
	errorCheck(err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var tmp bson.D
	err = collection.FindOne(ctx, bson.M{"login": logUser.Login}).Decode(&tmp)
	if err == mongo.ErrNoDocuments {
		c.IndentedJSON(http.StatusNotFound, myError{"Username does not exist"})
		return
	}
	errorCheck(err)
	var userDB user

	userDB._ID = tmp[0].Value.(primitive.ObjectID)
	userDB.Login = tmp[1].Value.(string)
	userDB.Password = tmp[2].Value.(string)

	if userDB.Password != logUser.Password {
		c.IndentedJSON(http.StatusNotFound, myError{"Invalid password"})
		return
	}

	expiration := time.Now().Add(3 * 24 * time.Hour)
	cookie := http.Cookie{Name: "csrftoken", Value: userDB._ID.Hex(), Expires: expiration}

	http.SetCookie(c.Writer, &cookie)
	c.IndentedJSON(http.StatusOK, userIdResponse{userDB._ID.Hex()})

	//cok, err := c.Request.Cookie("csrftoken")
	//fmt.Println(cok)
}

func depositAsset(c *gin.Context) {
	cookie, err := c.Request.Cookie("csrftoken")
	errorCheck(err)

	var newDeposit depositRequest
	err = c.BindJSON(&newDeposit)
	errorCheck(err)

	values := map[string]string{"name": newDeposit.Asset}
	json_data, err := json.Marshal(values)
	if err != nil { log.Fatal(err) }
	resp, err := http.Post("http://localhost:8001/assets/fiat", "application/json", bytes.NewBuffer(json_data))
	if err != nil { log.Fatal(err) }

	var res map[string]interface{}

	json.NewDecoder(resp.Body).Decode(&res)

	if(newDeposit.Amount >= 2000) {
		c.IndentedJSON(http.StatusBadRequest, myError{"The deposit is too big (maximum 2000)"})
		return
	}
	if(newDeposit.Amount <= 0) {
		c.IndentedJSON(http.StatusBadRequest, myError{"The deposit is too small"})
		return
	}
	if(!res["result"].(bool)) {
		c.IndentedJSON(http.StatusBadRequest, myError{"Currency is not fiat"})
		return
	}

	//asset service
	//There is something very crooked and bad.
	//0 - EUR
	//1 - ETH

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var tmp bson.D
	objId, err := primitive.ObjectIDFromHex(cookie.Value)
	filter := bson.M{"_id": bson.M{"$eq": objId}}
	if err != nil { log.Fatal(err) }
	err = collection.FindOne(ctx, filter).Decode(&tmp)
	if err != nil { log.Fatal(err) }

	var assetsUpdate []float64
	assetsUpdate = append(assetsUpdate, tmp[4].Value.(primitive.A)[0].(float64) + newDeposit.Amount)
	assetsUpdate = append(assetsUpdate, tmp[4].Value.(primitive.A)[1].(float64))


	update := bson.D{{Key: "$set", Value: bson.D{{Key: "assets", Value: assetsUpdate}}}}
	_, err = collection.UpdateOne(ctx, filter, update) //updatebyid doesnt work .-.
	if err != nil { log.Fatal(err) }

	eventRequest := eventDepositRequest{"NEW DEPOSIT", cookie.Value, "EUR", newDeposit.Amount}
	json_data, err = json.Marshal(eventRequest)
	if err != nil { c.IndentedJSON(http.StatusNotAcceptable, err); return }
	_, err = http.Post("http://localhost:8002/event/deposit", "application/json", bytes.NewBuffer(json_data)) //sending request to event service


	var depResp depositResponse
	depResp = depositResponse{cookie.Value, assetsUpdate[0], assetsUpdate[1]}
	c.IndentedJSON(http.StatusOK, depResp)
}

func assetsChanging(c *gin.Context) {
	var newAsset eventDepositRequest

	err := c.BindJSON(&newAsset)
	if err != nil { log.Fatal(err); return }

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var tmp bson.D
	objId, err := primitive.ObjectIDFromHex(newAsset.UserId)
	filter := bson.M{"_id": bson.M{"$eq": objId}}
	if err != nil { log.Fatal(err) }
	err = collection.FindOne(ctx, filter).Decode(&tmp)
	if err != nil { log.Fatal(err) }

	var assetsUpdate []float64
	assetsUpdate = append(assetsUpdate, tmp[4].Value.(primitive.A)[0].(float64))
	assetsUpdate = append(assetsUpdate, tmp[4].Value.(primitive.A)[1].(float64))

	if(newAsset.Asset == "EUR") {
		assetsUpdate[0] -= newAsset.Amount
	} else if(newAsset.Asset == "ETH") {
		assetsUpdate[1] -= newAsset.Amount
	}

	update := bson.D{{Key: "$set", Value: bson.D{{Key: "assets", Value: assetsUpdate}}}}
	_, err = collection.UpdateOne(ctx, filter, update)
	if err != nil { log.Fatal(err) }

	c.IndentedJSON(http.StatusOK, assetsUpdate)
}

func isAdmin(c *gin.Context) {
	var checkId userIdResponse
	err := c.BindJSON(&checkId)

	objId, err := primitive.ObjectIDFromHex(checkId.UserId)
	if err != nil { log.Fatal(err); return }
	filter := bson.M{"_id": bson.M{"$eq": objId}}

	var res user
	res.IsAdmin = false
	collection.FindOne(context.Background(), filter).Decode(&res)
	fmt.Println(res)
	c.IndentedJSON(http.StatusOK, res)
}

func checkAssets(c *gin.Context) {
	var checkId userIdResponse
	err := c.BindJSON(&checkId)
	if(err != nil) { log.Fatal(err); return }

	objId, err := primitive.ObjectIDFromHex(checkId.UserId)
	if err != nil { log.Fatal(err); return }
	filter := bson.M{"_id": bson.M{"$eq": objId}}

	var tmp bson.D
	var res depositResponse
	collection.FindOne(context.Background(), filter).Decode(&tmp)

	res.UserId = checkId.UserId
	res.EUR = tmp[4].Value.(primitive.A)[0].(float64)
	res.ETH = tmp[4].Value.(primitive.A)[1].(float64)

	c.IndentedJSON(http.StatusOK, res)
}

func checkCookie(c *gin.Context) {
	cok, err := c.Request.Cookie("csrftoken")
	errorCheck(err)
	c.IndentedJSON(http.StatusOK, userIdResponse{cok.Value})
}

func main() {
	router := gin.Default()
	//replace password before git commit
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

	err = client.Ping(ctx, readpref.Primary())
	errorCheck(err)

	databases, err := client.ListDatabaseNames(ctx, bson.M{})
	errorCheck(err)

	fmt.Println(databases)

	collection = client.Database("mongo").Collection("users")

	//var nw = user{Login: "Vasja", Password: "ranbluat"}
	//res, err := collection.InsertOne(context.Background(), nw)
	//id := res.InsertedID

	//fmt.Println(id)

	router.POST("/assets/change", assetsChanging)
	router.GET("/login", loginUser)
	router.GET("/users", getUsers)
	router.POST("/register", createUser)
	router.POST("/deposit", depositAsset)
	router.GET("/check", checkCookie)
	router.POST("/admin", isAdmin)
	router.POST("/assets", checkAssets)
	router.Run("localhost:8000")
}
