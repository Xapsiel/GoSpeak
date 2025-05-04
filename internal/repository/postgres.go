package repository

import (
	"fmt"

	"GoSpeak/internal/config"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func NewPostgresDB(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s dbname=postgres password='%s' sslmode=%s", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.SSLMode)
	initialDB, err := sqlx.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("Ошибка подключения к базе данных postgres")
	}
	_, err = initialDB.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.Name))
	if err != nil {
		err = nil
	}
	defer func() {
		initialDB.Close()
	}()
	mainConnStr := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password='%s' sslmode=%s", cfg.Host, cfg.Port, cfg.User, cfg.Name, cfg.Password, cfg.SSLMode)
	db, err := sqlx.Open("postgres", mainConnStr)
	if err != nil {
		return nil, fmt.Errorf("Ошибка подключения к базе данных %s", cfg.Name)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("Ошибка пингования базы данных")
	}
	err = Migrate(db, cfg)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func Migrate(db *sqlx.DB, cfg config.DatabaseConfig) error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("Ошибка создания объекта драйвера")
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		cfg.Name, driver,
	)
	if err != nil {
		return fmt.Errorf("Ошибка создания объекта миграция")
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("Ошибка принятия миграции")
	}
	return nil
}
