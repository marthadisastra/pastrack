package main

import (
	"github.com/beego/beego/v2/server/web"
	"log"
	"net/http"
	"pastrack/web/controllers"
	"pastrack/web/models"
	"pastrack/web/routers"
	"time"
)

func main() {
	database, e := models.Open()
	if e != nil {
		log.Fatal(e)
	}
	defer database.Close()
	controllers.Configure(database)
	routers.Register()
	controllers.StartBackground()
	server := &http.Server{Addr: models.Env("WEB_ADDR", "127.0.0.1:18089"), Handler: controllers.Security(web.BeeApp.Handlers), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 90 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	log.Printf("Pastrack Beego MVC listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
