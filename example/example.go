package example

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/mmbednarek/smartapi"
)

var (
	ErrUserDoesNotExists = errors.New("user does not exists")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type Storage interface {
	StoreUser(id string, data *UserData) error
	GetUser(id string) (*UserData, error)
}

type UserData struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type API struct {
	*smartapi.Server
	storage Storage
}

func NewAPI(storage Storage) *API {
	return &API{
		Server:  smartapi.NewServer(nil),
		storage: storage,
	}
}

type Order struct {
}

func (a *API) Init() {
	a.Get("/user", a.GetUser,
		smartapi.Context(),
		smartapi.QueryParam("user"),
	)

	a.Post("/user/{user}", a.NewUser,
		smartapi.URLParam("user"),
		smartapi.JSONBody(UserData{}),

		smartapi.ResponseStatus(http.StatusCreated),
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
		smartapi.Middleware(middleware.DefaultLogger),
	)

	a.Get("/order/{id}", func(orderID string) (*Order, error) {
		var order *Order
		ErrNoSuchOrder := errors.New("no such order")
		err := ErrNoSuchOrder
		if err != nil {
			if errors.Is(err, ErrNoSuchOrder) {
				return nil, smartapi.WrapError(http.StatusNotFound, err, "no such order")
			}
			return nil, err
		}
		return order, nil
	},
		smartapi.URLParam("id"),
	)
}

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

func (a *API) NewUser(userID string, userData *UserData) error {
	if err := a.storage.StoreUser(userID, userData); err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			return smartapi.WrapError(http.StatusBadRequest, err, "user already exists")
		}
		return err
	}
	return nil
}
