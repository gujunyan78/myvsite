package main

import (
	"log"

	"myvsite/internal/config"
	"myvsite/internal/handler"
	"myvsite/internal/model"
	"myvsite/internal/router"
)

func main() {
	c := config.Load()

	if err := model.Init(c); err != nil {
		log.Fatalf("Initialization failed: %v", err)
	}
	if err := handler.SetUploadDir(c.UploadDir); err != nil {
		log.Fatalf("Failed to create upload dir: %v", err)
	}

	r := router.New(c)

	log.Printf("myvsite started, listening on http://%s (site: / , admin: /admin)", c.Addr)
	if err := r.Run(c.Addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
