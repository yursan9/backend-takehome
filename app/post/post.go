package post

import (
	"app/repository"
	"app/server"
	"app/user"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
)

type Post struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	AuthorID  int    `json:"author_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Comment struct {
	Author    string `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

type PostsParam struct {
	AuthorID int
	PaginationParam
}

type PaginationParam struct {
	Page int
	Size int
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) PostHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		idPath := r.PathValue("id")
		id, err := strconv.Atoi(idPath)
		if err != nil {
			server.ErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		p := s.Post(r.Context(), id)
		if p == nil {
			server.ErrorResponse(w, http.StatusNotFound, ErrNotFound)
			return
		}

		server.JSONResponse(w, http.StatusOK, p)
	}
}

func (s *Service) Post(ctx context.Context, id int) *Post {
	repo := repository.New(s.db)
	p := repo.Post(ctx, id)
	if p == nil {
		return nil
	}

	res := mapPostRepoToService(*p)
	return &res
}

func (s *Service) PostsHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		urlParams := r.URL.Query()
		var errs []error

		authorIDStr := urlParams.Get("author_id")
		authorID := 0
		if authorIDStr != "" {
			var err error
			authorID, err = strconv.Atoi(authorIDStr)
			if err != nil {
				errs = append(errs, err)
			}
		}

		pageStr := urlParams.Get("page")
		page := 1
		if pageStr != "" {
			var err error
			page, err = strconv.Atoi(pageStr)
			if err != nil {
				page = 1
				errs = append(errs, err)
			}
		}

		sizeStr := urlParams.Get("size")
		size := 10
		if sizeStr != "" {
			var err error
			size, err = strconv.Atoi(sizeStr)
			if err != nil {
				size = 10
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			server.ErrorResponse(w, http.StatusBadRequest, errors.Join(errs...))
			return
		}

		params := PostsParam{
			AuthorID: authorID,
			PaginationParam: PaginationParam{
				Page: page,
				Size: size,
			},
		}

		ps, total := s.Posts(r.Context(), params)
		output := struct {
			Next  string
			Prev  string
			Total int
			Data  []Post
		}{
			Total: total,
			Data:  ps,
		}
		server.JSONResponse(w, http.StatusOK, output)
	}
}

func (s *Service) Posts(ctx context.Context, param PostsParam) ([]Post, int) {
	repo := repository.New(s.db)
	repoParam := repository.PostsParam{
		AuthorID: param.AuthorID,
	}
	repoParam.Page = param.Page
	repoParam.Size = param.Size
	ps, total := repo.Posts(ctx, repoParam)

	res := make([]Post, 0, len(ps))
	for _, p := range ps {
		res = append(res, mapPostRepoToService(p))
	}

	return res, total
}

func (s *Service) CreatePostHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		authorID := user.IDFromContext(r.Context())

		var input Post
		json.NewDecoder(r.Body).Decode(&input)
		input.AuthorID = authorID

		err := s.CreatePost(r.Context(), input)
		if err != nil {
			server.ErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) CreatePost(ctx context.Context, data Post) error {
	err := s.execInTx(ctx, func(r *repository.Repository) error {
		u := r.User(ctx, data.AuthorID)
		if u == nil {
			return ErrNotFound
		}

		return r.CreatePost(ctx, repository.Post{
			AuthorID: u.ID,
			Title:    data.Title,
			Content:  data.Content,
		})
	})

	return err
}

func (s *Service) UpdatePostHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		idPath := r.PathValue("id")
		id, err := strconv.Atoi(idPath)
		if err != nil {
			server.ErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		authorID := user.IDFromContext(r.Context())

		var input Post
		json.NewDecoder(r.Body).Decode(&input)
		input.AuthorID = authorID

		err = s.UpdatePost(r.Context(), id, input)
		if err != nil {
			server.ErrorResponse(w, http.StatusNotFound, ErrNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) UpdatePost(ctx context.Context, id int, data Post) error {
	err := s.execInTx(ctx, func(r *repository.Repository) error {
		p := r.Post(ctx, id)
		fmt.Println(p)
		fmt.Println(data)
		if p == nil {
			return ErrNotFound
		}

		if p.AuthorID != data.AuthorID {
			return ErrNotFound
		}

		return r.UpdatePost(ctx, p.ID, repository.Post{
			AuthorID: data.AuthorID,
			Title:    data.Title,
			Content:  data.Content,
		})
	})

	return err
}

func (s *Service) DeletePostHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		idPath := r.PathValue("id")
		id, err := strconv.Atoi(idPath)
		if err != nil {
			server.ErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		authorID := user.IDFromContext(r.Context())

		err = s.DeletePost(r.Context(), id, authorID)
		if err != nil {
			server.ErrorResponse(w, http.StatusNotFound, ErrNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) DeletePost(ctx context.Context, id int, authorId int) error {
	err := s.execInTx(ctx, func(r *repository.Repository) error {
		p := r.Post(ctx, id)
		if p == nil {
			return ErrNotFound
		}

		if p.AuthorID != authorId {
			return ErrNotFound
		}

		return r.DeletePost(ctx, p.ID, p.AuthorID)
	})

	return err
}

func (s *Service) execInTx(ctx context.Context, fn func(*repository.Repository) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	repo := repository.New(tx)
	repo.ForUpdate = true
	err = fn(repo)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

func mapPostRepoToService(data repository.Post) Post {
	return Post{
		ID:        data.ID,
		Title:     data.Title,
		Content:   data.Content,
		AuthorID:  data.AuthorID,
		CreatedAt: data.CreatedAt.Format(time.DateTime),
		UpdatedAt: data.UpdatedAt.Format(time.DateTime),
	}
}
