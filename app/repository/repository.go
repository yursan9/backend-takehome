package repository

import (
	"context"
	"database/sql"

	"github.com/georgysavva/scany/v2/dbscan"
)

type DB interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

type Repository struct {
	db DB
}

type User struct {
	Name         string
	Email        string
	PasswordHash string
}

func New(db DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) User(ctx context.Context, id int) *User {
	sqlQuery := "SELECT * FROM user WHERE id = $1 LIMIT 1"
	rows, err := r.db.QueryContext(ctx, sqlQuery, id)
	if err != nil {
		return nil
	}

	var res User
	dbscan.ScanRow(&res, rows)
	return &res
}

func (r *Repository) UserWithEmail(ctx context.Context, email string) *User {
	sqlQuery := "SELECT * FROM user WHERE email = $1 LIMIT 1"
	rows, err := r.db.QueryContext(ctx, sqlQuery, email)
	if err != nil {
		return nil
	}

	var res User
	dbscan.ScanRow(&res, rows)
	return &res
}

func (r *Repository) CreateUser(ctx context.Context, data User) error {
	sqlQuery := "INSERT INTO user (name, email, password_hash) VALUES($1, $2, $3)"
	_, err := r.db.ExecContext(ctx, sqlQuery, data.Name, data.Email, data.PasswordHash)
	if err != nil {
		return err
	}
	return nil
}
