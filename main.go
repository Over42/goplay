package main

import (
	"goplay/config"
	"goplay/handler"
	"goplay/matchmaker"
	"goplay/repository"

	"log"
)

func main() {
	cfg := config.NewConfig()

	db, err := repository.OpenDatabase(cfg)
	if err != nil {
		log.Fatalf("Could not connect to DB: %s", err)
	}
	defer db.Close()

	rep := repository.NewSQLRepository(db)
	mm := matchmaker.NewMatchmaker(rep, cfg, matchmaker.RequestServer)
	hdl := handler.NewHttpHandler(mm)

	handler.StartRouter(cfg, hdl)
}
