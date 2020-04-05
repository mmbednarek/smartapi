package example

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/mmbednarek/smartapi"
)

var (
	// ErrUserDoesNotExists is used when when the user doesn't exists
	ErrUserDoesNotExists = errors.New("user does not exists")
	// ErrUserAlreadyExists is used when when user already exists
	ErrUserAlreadyExists = errors.New("user already exists")
)

// Storage is a storage interface
type Storage interface {
	StoreUser(id string, data *UserData) error
	GetUser(id string) (*UserData, error)
}

// UserData contains basic user information
type UserData struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// API struct
type API struct {
	*smartapi.Server
	storage Storage
}

// NewAPI constructor
func NewAPI(storage Storage) *API {
	return &API{
		Server:  smartapi.NewServer(nil),
		storage: storage,
	}
}

// Order structure
type Order struct {
}

// Init inits the api
func (a *API) Init() {
	a.With(middleware.DefaultLogger).Route("/user", func(r smartapi.Router) {
		r.Get("/", a.GetUser,
			smartapi.QueryParam("user"),
		)

		r.Post("/{user}", a.NewUser,
			smartapi.URLParam("user"),
			smartapi.JSONBody(UserData{}),
			smartapi.ResponseStatus(http.StatusCreated),
		)
	},
		smartapi.Context(),
	)

	a.Get("/test", func(name string, cookies smartapi.Cookies, headers smartapi.Headers) error {
		cookies.Add(&http.Cookie{
			Name:  "Session-Token",
			Value: "token",
		})
		return nil
	},
		smartapi.Cookie("Session"),
		smartapi.ResponseCookies(),
		smartapi.ResponseHeaders(),
	)
}

// GetUser handles the user endpoint
func (a *API) GetUser(ctx context.Context, name string) (*UserData, error) {
	user, err := a.storage.GetUser(name)
	if err != nil {
		if errors.Is(err, ErrUserDoesNotExists) {
			return nil, smartapi.WrapError(http.StatusNotFound, err, "user does not exists")
		}
		return nil, err
	}
	return user, nil
}

// NewUser handles the POST user endpoint
func (a *API) NewUser(ctx context.Context, userID string, userData *UserData) error {
	if err := a.storage.StoreUser(userID, userData); err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			return smartapi.WrapError(http.StatusBadRequest, err, "user already exists")
		}
		return err
	}
	return nil
}
