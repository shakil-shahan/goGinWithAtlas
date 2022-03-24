package main

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"goGinWithAtlas/database"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	_ "github.com/heroku/x/hmetrics/onload"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

//this object/struct determines what a task object should look like.
type Task struct {
	ID          primitive.ObjectID `bson:"_id"`
	Name        *string            `json:"name" validate:"required,min=2,max=100"`
	Description *string            `json:"description" validate:"required,min=0,max=200"`
	IsCompleted bool               `json:"isCompleted"`
	Parent_id   string             `json:"parent_id"`
	Task_date   time.Time          `json:"task_date"`
	Created_at  time.Time          `json:"created_at"`
	Updated_at  time.Time          `json:"updated_at"`
}

// create a validator object
var validate = validator.New()

//this function rounds the price value down to 2 decimal places
func ToFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(Round(num*output)) / output
}
func Round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

//connect to to the database and open a task collection
var taskCollection *mongo.Collection = database.OpenCollection(database.Client, "tasks")

func main() {

	port := os.Getenv("PORT")

	if port == "" {
		port = "8000"
	}

	router := gin.New()
	router.Use(gin.Logger())

	// this is the create task endpoint
	router.POST("/task", func(ctx *gin.Context) {
		//declare a variable of type task
		var task Task

		//bind the object that comes in with the declared varaible. thrrow an error if one occurs
		if err := ctx.BindJSON(&task); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// use the validation packge to verify that all items coming in meet the requirements of the struct
		validationErr := validate.Struct(task)
		if validationErr != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		// assing the time stamps upon creation
		task.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		task.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		task.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		//generate new ID for the object to be created
		task.ID = primitive.NewObjectID()

		// assign the the auto generated ID to the primary key attribute
		task.Parent_id = task.ID.Hex()

		// make the task completion by default false
		task.IsCompleted = false

		//insert the newly created object into mongodb
		result, insertErr := taskCollection.InsertOne(context.Background(), task)
		if insertErr != nil {
			msg := fmt.Sprintf("Task was not created")
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		//return the id of the created object to the frontend
		ctx.JSON(http.StatusOK, result)

	})

	// Get All tasks
	router.GET("/task", func(ctx *gin.Context) {
		var tasks []Task

		results, err := taskCollection.Find(ctx, bson.M{})

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		//reading from the db in an optimal way
		defer results.Close(ctx)
		for results.Next(ctx) {
			var singleTask Task
			if err = results.Decode(&singleTask); err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}

			tasks = append(tasks, singleTask)
		}

		//return all the tasks
		ctx.JSON(http.StatusOK, tasks)
	})

	// Get a single task
	router.GET("/task/:taskId", func(ctx *gin.Context) {
		taskId := ctx.Param("taskId")

		var task Task

		objId, _ := primitive.ObjectIDFromHex(taskId)

		err := taskCollection.FindOne(ctx, bson.M{"_id": objId}).Decode(&task)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		//return the task
		ctx.JSON(http.StatusOK, task)
	})

	//this runs the server and allows it to listen to requests.
	router.Run(":" + port)
}
