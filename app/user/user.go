package user

import (
	"app/repository"
	"app/server"
	"app/session"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAlreadyRegistered = errors.New("already registered")
	ErrInvalidLogin      = errors.New("invalid login")
	ErrNotFound          = errors.New("not found")
)

type User struct {
	Name     string
	Email    string
	Password string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) RegisterHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			server.ErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		err = s.Register(r.Context(), input.Name, input.Email, input.Password)
		if err != nil {
			server.ErrorResponse(w, http.StatusUnprocessableEntity, err)
			return
		}

		w.WriteHeader(200)
	}
}

func (s *Service) Register(ctx context.Context, name, email, password string) error {
	err := s.execInTx(ctx, func(r *repository.Repository) error {
		u := r.UserWithEmail(ctx, email)
		if u != nil {
			return fmt.Errorf("email %s: %w", email, ErrAlreadyRegistered)
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("error register: %w", err)
		}

		return r.CreateUser(ctx, repository.User{
			Name:         name,
			Email:        email,
			PasswordHash: string(passwordHash),
		})
	})

	return err
}

func (s *Service) LoginHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			server.ErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		token, err := s.Login(r.Context(), input.Email, input.Password)
		if err != nil {
			server.ErrorResponse(w, http.StatusUnprocessableEntity, err)
			return
		}

		output := struct {
			Token string `json:"token"`
		}{
			Token: token,
		}
		server.JSONResponse(w, 200, output)
	}
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	repo := repository.New(s.db)
	u := repo.UserWithEmail(ctx, email)
	if u == nil {
		return "", fmt.Errorf("user with email %s: %w", email, ErrNotFound)
	}

	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", fmt.Errorf("user password not match: %w", ErrInvalidLogin)
	}

	token := session.Create(u.ID)
	return token, nil
}

func (s *Service) execInTx(ctx context.Context, fn func(*repository.Repository) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	repo := repository.New(tx)
	err = fn(repo)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}
