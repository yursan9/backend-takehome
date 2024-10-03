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
	ErrNotFound      = errors.New("not found")
	ErrNotAuthorized = errors.New("not authorized")
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
	Content   string `json:"content"`
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
			return fmt.Errorf("invalid author with id %d: %w", data.AuthorID, ErrNotFound)
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
			status := http.StatusInternalServerError
			switch {
			case errors.Is(err, ErrNotAuthorized):
				status = http.StatusForbidden
			case errors.Is(err, ErrNotFound):
				status = http.StatusNotFound
			}
			server.ErrorResponse(w, status, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) UpdatePost(ctx context.Context, id int, data Post) error {
	err := s.execInTx(ctx, func(r *repository.Repository) error {
		p := r.Post(ctx, id)
		if p == nil {
			return fmt.Errorf("unable to update post with id %d: %w", id, ErrNotFound)
		}

		if p.AuthorID != data.AuthorID {
			return fmt.Errorf("unable to update post with id %d: %w", id, ErrNotAuthorized)
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
			status := http.StatusInternalServerError
			switch {
			case errors.Is(err, ErrNotAuthorized):
				status = http.StatusForbidden
			case errors.Is(err, ErrNotFound):
				status = http.StatusNotFound
			}
			server.ErrorResponse(w, status, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) DeletePost(ctx context.Context, id int, authorId int) error {
	err := s.execInTx(ctx, func(r *repository.Repository) error {
		p := r.Post(ctx, id)
		if p == nil {
			return fmt.Errorf("unable to delete post with id %d: %w", id, ErrNotFound)
		}

		if p.AuthorID != authorId {
			return fmt.Errorf("unable to delete post with id %d: %w", id, ErrNotAuthorized)
		}

		return r.DeletePost(ctx, p.ID, p.AuthorID)
	})

	return err
}

func (s *Service) CommentsHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		idPath := r.PathValue("id")
		id, err := strconv.Atoi(idPath)
		if err != nil {
			server.ErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		cs := s.Comments(r.Context(), id)

		output := struct {
			PostID   int       `json:"post_id"`
			Comments []Comment `json:"comments"`
		}{
			PostID:   id,
			Comments: cs,
		}
		server.JSONResponse(w, http.StatusOK, output)
	}
}

func (s *Service) Comments(ctx context.Context, postID int) []Comment {
	repo := repository.New(s.db)
	cs := repo.Comments(ctx, postID)

	res := make([]Comment, 0, len(cs))
	for _, c := range cs {
		res = append(res, mapCommentRepoToService(c))
	}
	return res
}

func (s *Service) CreateCommentHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		idPath := r.PathValue("id")
		id, err := strconv.Atoi(idPath)
		if err != nil {
			server.ErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		var input Comment
		json.NewDecoder(r.Body).Decode(&input)

		err = s.CreateComment(r.Context(), id, input)
		if err != nil {
			server.ErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (s *Service) CreateComment(ctx context.Context, postID int, data Comment) error {
	err := s.execInTx(ctx, func(r *repository.Repository) error {
		p := r.Post(ctx, postID)
		if p == nil {
			return ErrNotFound
		}

		return r.CreateComment(ctx, postID, repository.Comment{
			PostID:     p.ID,
			AuthorName: data.Author,
			Content:    data.Content,
		})
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

func mapCommentRepoToService(data repository.Comment) Comment {
	return Comment{
		Author:    data.AuthorName,
		Content:   data.Content,
		CreatedAt: data.CreatedAt.Format(time.DateTime),
	}
}
