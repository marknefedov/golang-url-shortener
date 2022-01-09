package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CreateUrlRequest struct {
	End_url string `json:"end_url" bson:"end_url"`
}

type RedirectRule struct {
	End_url  string `json:"end_url" bson:"end_url"`
	From_url string `json:"from_url" bson:"from_url"`
}

func main() {
	mongo_string := os.Getenv("MONGO_STRING")
	base_url := os.Getenv("BASE_URL")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongo_string))
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	redirect_rules := client.Database("myFirstDatabase").Collection("redirect_rules")
	redirects := client.Database("myFirstDatabase").Collection("redirects")

	r := gin.Default()
	r.GET("/r/:redir", func(c *gin.Context) {
		redir := c.Param("redir")
		var redirect_rule RedirectRule
		err := redirect_rules.FindOne(ctx, bson.D{{"from_url", redir}}).Decode(&redirect_rule)
		if err != nil {
			log.Println(err.Error())
			c.String(404, "Not found")
		}
		redirects.InsertOne(ctx, bson.D{{"from", redir}, {"end_url", redirect_rule.End_url}, {"ip", c.ClientIP()}, {"headers", c.Request.Header}})
		c.Redirect(301, redirect_rule.End_url)
	})
	r.POST("/api/create_url", func(c *gin.Context) {
		var create_url_request CreateUrlRequest
		c.BindJSON(&create_url_request)
	rerandom:
		rand_string := randStr(6)
		var redirect_rule RedirectRule
		err := redirect_rules.FindOne(ctx, bson.D{{"from_url", rand_string}}).Decode(&redirect_rule)
		if err == nil {
			goto rerandom
		}
		redirect_rule.From_url = rand_string
		redirect_rule.End_url = create_url_request.End_url
		redirect_rules.InsertOne(ctx, redirect_rule)
		type Response struct {
			Redirect_url string `json:"redirect_url"`
		}
		c.JSON(200, Response{Redirect_url: base_url + rand_string})
	})
	r.Run("0.0.0.0:3030")
}

func randStr(len int) string {
	buff := make([]byte, len)
	rand.Read(buff)
	str := base64.StdEncoding.EncodeToString(buff)
	// Base 64 can be longer than len
	return str[:len]
}
