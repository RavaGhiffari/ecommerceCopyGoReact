package main

import (
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"goFrameworkFirstTry/data"
	"goFrameworkFirstTry/handler"
	"goFrameworkFirstTry/repository"
)

func main() {
	// 1. Initialize data source
	users := data.GetUsers()

	// 2. Inject data into repository
	userRepo := repository.NewUserRepository(users)

	// 3. Inject repository into handler
	userHandler := handler.NewUserHandler(userRepo)

	// 4. Register routes
	router := httprouter.New()

	router.GET("/", Index)
	router.GET("/hello/:name", Hello)
	router.GET("/health/:name", Health)

	router.GET("/api/users", userHandler.GetUsers)
	router.GET("/api/users/:id", userHandler.GetUserByID)

	log.Fatal(http.ListenAndServe(":8081", router))
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Write([]byte("Welcome!\n"))
}

func Hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Write([]byte("hello, " + ps.ByName("name") + "!\n"))
}

func Health(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Write([]byte(ps.ByName("name") + "'s system is good to go!"))
}
