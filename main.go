package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/acme/autocert"
)

type Task struct {
	ID          int    `json:"id"`
	Description string `json:"label" binding:"required"`
	Count       int    `json:"count"`
}

type Event struct {
	TaskID      int       `json:"taskid" binding:"required"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

type TaskDatabase struct {
	Tasks  map[int]Task `json:"tasks"`
	NextID int          `json:"nextid"`
	Events []Event      `json:"events"`
}

type IDRequest struct {
	ID int `uri:"id"`
}

func (db *TaskDatabase) ListTaskHandler(ctx *gin.Context) {
	var tasklist []Task = make([]Task, len(db.Tasks))

	for _, task := range db.Tasks {
		tasklist = append(tasklist, task)
	}
	ctx.JSON(200, tasklist)
}

func (db *TaskDatabase) TakeTaskHandler(ctx *gin.Context) {
	var ir IDRequest
	if err := ctx.ShouldBindUri(&ir); err != nil {
		ctx.JSON(400, gin.H{"msg": err})
		return
	}

	t, ok := db.Tasks[ir.ID]
	if !ok {
		ctx.JSON(404, gin.H{"msg": "Object not found."})
		return
	}
	ctx.JSON(200, t)
	return
}

func (db *TaskDatabase) CreateTaskHandler(ctx *gin.Context) {
	t := Task{}
	if err := ctx.ShouldBindJSON(&t); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if t.ID > db.NextID { // id given is larger than previous maxid
		db.NextID = t.ID + 1
		println("id was larger, setting nextid")
		println(db.NextID)
	}
	if t.ID == 0 { // no id given
		t.ID = db.NextID
		db.NextID = db.NextID + 1
		println("no id was given, setting taskid to nextid")
		println(db.NextID)
	}
	db.Tasks[t.ID] = t
	ctx.JSON(http.StatusAccepted, t)

	println("create")
}

func (db *TaskDatabase) DeleteTaskHandler(ctx *gin.Context) {
	var ir IDRequest
	println("delete")
	if err := ctx.ShouldBindUri(&ir); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	if _, ok := db.Tasks[ir.ID]; !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"msg": "Object not found."})
		return
	}

	delete(db.Tasks, ir.ID)
	ctx.JSON(http.StatusAccepted, gin.H{"msg:": "Object deleted."})
	println("delete ok")
}

func (db *TaskDatabase) CreateEventHandler(ctx *gin.Context) {
	var event Event
	var task Task
	println("create task")
	if err := ctx.ShouldBindJSON(&event); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	task, ok := db.Tasks[event.TaskID]
	if !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"msg": "Event not found."})
		return
	}
	event.Timestamp = time.Now()
	event.Description = task.Description
	db.Events = append(db.Events, event)
	task.Count = task.Count + 1
	println(task.Count)
	ctx.JSON(http.StatusAccepted, event)

}

func (db *TaskDatabase) ListHandler(ctx *gin.Context) {
	ctx.JSON(200, db)
}

func (db *TaskDatabase) CreateHandler(ctx *gin.Context) {
	database := TaskDatabase{}
	if err := ctx.ShouldBindJSON(&database); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.Tasks = database.Tasks
	db.Events = database.Events
	db.NextID = database.NextID
	ctx.JSON(http.StatusAccepted, db)

	println("create")
}

func (db *TaskDatabase) ListEventHandler(ctx *gin.Context) {
	var eventlist []Event = make([]Event, len(db.Events))

	for _, event := range db.Events {
		eventlist = append(eventlist, event)
	}
	ctx.JSON(200, eventlist)
}

func (db *TaskDatabase) SinglePageHandler(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title": "Main website",
	})
}

func main() {
	var tlsDomain string

	flag.StringVar(&tlsDomain, "t", "", "Domain for tls, or empty for no tls.")

	db := TaskDatabase{Tasks: make(map[int]Task), NextID: 1, Events: make([]Event, 0, 10000)}
	r := gin.Default()

	r.LoadHTMLGlob("./*.tmpl")

	apiGroup := r.Group("/api")

	apiGroup.POST("task", db.CreateTaskHandler)
	apiGroup.GET("task", db.ListTaskHandler)
	apiGroup.GET("task/:id", db.TakeTaskHandler)
	apiGroup.DELETE("task/:id", db.DeleteTaskHandler)

	apiGroup.POST("event", db.CreateEventHandler)
	apiGroup.GET("event", db.ListEventHandler)

	apiGroup.GET("db", db.ListHandler)
	apiGroup.POST("db", db.CreateHandler)

	r.Group("/").GET("/", db.SinglePageHandler)

	r.GET("/routes", func(ctx *gin.Context) {
		if gin.Mode() == gin.DebugMode {
			ctx.Data(200, "application/json; charset=utf-8", []byte(fmt.Sprintf("%v", r.Routes())))
		}
	})
	if tlsDomain != "" {
		m := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(tlsDomain),
			Cache:      autocert.DirCache("/var/www/.cache"),
		}
		log.Fatal(autotls.RunWithManager(r, &m))
	}
	if tlsDomain == "" {
		r.Run(":8080")

	}
}
