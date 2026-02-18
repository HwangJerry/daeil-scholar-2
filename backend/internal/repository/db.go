package repository

import (
	"github.com/dflh-saf/backend/internal/config"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func NewDB(cfg config.DBConfig) (*sqlx.DB, error) {
	db, err := sqlx.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
