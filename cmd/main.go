package main

import (
	"flag"
	"log/slog"
	"os"

	"GoSpeak/internal/config"
	"GoSpeak/internal/entrypoints/http"
	"GoSpeak/internal/repository"
	"GoSpeak/internal/service"

	"github.com/gofiber/fiber/v2"
)

func main() {
	configPath := flag.String("c", "config/config.yaml", "The path to the config file")
	flag.Parse()
	config, err := config.New(*configPath)

	if err != nil {
		slog.Error("unable to read config:", err.Error())
		os.Exit(1)
	}
	db, err := repository.NewPostgresDB(config.DatabaseConfig)
	if err != nil {
		slog.Error("unable to connect to database:", err.Error())
		os.Exit(1)
	}
	slog.Info("connected to database")
	repos := repository.NewRepository(db)
	slog.Info("initializing repos")
	services := service.NewService(*repos)
	router := http.NewRouter(*services, config.HostConfig)
	app := fiber.New(fiber.Config{})

	router.Routes(app)
	go router.HandleWebSocketMessage()

	slog.Error(app.Listen(":" + config.HostConfig.Port).Error())

}
