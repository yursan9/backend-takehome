package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/dbscan"
)

type DB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type Repository struct {
	db        DB
	ForUpdate bool
}

type User struct {
	ID           int    `db:"id"`
	Name         string `db:"name"`
	Email        string `db:"email"`
	PasswordHash string `db:"password_hash"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Post struct {
	ID        int    `db:"id"`
	AuthorID  int    `db:"author_id"`
	Title     string `db:"title"`
	Content   string `db:"content"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PostsParam struct {
	AuthorID int
	PaginationParam
}

type PaginationParam struct {
	Page int
	Size int
}

func New(db DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) User(ctx context.Context, id int) *User {
	sqlQuery := r.selectQuery(`SELECT * FROM user WHERE id = ? LIMIT 1`)
	rows, err := r.db.QueryContext(ctx, sqlQuery, id)
	if err != nil {
		slog.Error("failed to query user", "id", id, "err", err)
		return nil
	}

	var res User
	err = dbscan.ScanOne(&res, rows)
	if err != nil {
		slog.Error("failed to scan user", "id", id, "err", err)
		return nil
	}
	return &res
}

func (r *Repository) UserWithEmail(ctx context.Context, email string) *User {
	sqlQuery := r.selectQuery(`SELECT * FROM user WHERE email = ? LIMIT 1`)
	rows, err := r.db.QueryContext(ctx, sqlQuery, email)
	if err != nil {
		slog.Error("failed to query user", "email", email, "err", err)
		return nil
	}

	var res User
	err = dbscan.ScanOne(&res, rows)
	if err != nil {
		slog.Error("failed to scan user", "email", email, "err", err)
		return nil
	}
	return &res
}

func (r *Repository) CreateUser(ctx context.Context, data User) error {
	sqlQuery := `INSERT INTO user (name, email, password_hash) VALUES(?, ?, ?)`
	_, err := r.db.ExecContext(ctx, sqlQuery, data.Name, data.Email, data.PasswordHash)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) Post(ctx context.Context, id int) *Post {
	sqlQuery := r.selectQuery("SELECT * FROM post WHERE id = ? LIMIT 1")
	rows, err := r.db.QueryContext(ctx, sqlQuery, id)
	if err != nil {
		return nil
	}

	var res Post
	err = dbscan.ScanOne(&res, rows)
	if err != nil {
		return nil
	}
	return &res
}

func (r *Repository) Posts(ctx context.Context, param PostsParam) ([]Post, int) {
	if param.Page <= 0 {
		param.Page = 1
	}

	if param.Size <= 0 {
		param.Size = 10
	}

	sqlQuery := r.selectQuery("SELECT * FROM post")
	var args []any

	if param.AuthorID > 0 {
		sqlQuery += "WHERE author_id = ?"
		args = append(args, param.AuthorID)
	}

	total := r.count(ctx, sqlQuery, args...)
	sqlQuery = r.paginationQuery(sqlQuery, param.PaginationParam)
	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, 0
	}

	var res []Post
	dbscan.ScanAll(&res, rows)
	return res, total
}

func (r *Repository) CreatePost(ctx context.Context, data Post) error {
	sqlQuery := "INSERT INTO post (title, content, author_id) VALUES(?, ?, ?)"
	_, err := r.db.ExecContext(ctx, sqlQuery, data.Title, data.Content, data.AuthorID)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) UpdatePost(ctx context.Context, id int, data Post) error {
	sqlQuery := "UPDATE post SET title = ?, content = ? WHERE id = ? AND author_id = ?"
	_, err := r.db.ExecContext(ctx, sqlQuery, data.Title, data.Content, id, data.AuthorID)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) DeletePost(ctx context.Context, id int, authorID int) error {
	sqlQuery := "DELETE FROM post WHERE id = ? AND author_id = ?"
	_, err := r.db.ExecContext(ctx, sqlQuery, id, authorID)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) selectQuery(query string) string {
	if r.ForUpdate {
		query += " FOR UPDATE"
	}
	return query
}

func (r *Repository) count(ctx context.Context, query string, args ...any) int {
	query = strings.Replace(query, " * ", " COUNT(*) ", 1)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return 0
	}

	var total int
	dbscan.ScanOne(&total, rows)
	return total
}

func (r *Repository) paginationQuery(query string, param PaginationParam) string {
	limit := param.Size
	offset := (param.Page - 1) * param.Size

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", offset)
	}
	return query
}
