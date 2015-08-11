package restapi

import (
	"fmt"
	"strings"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
	"github.com/sebest/hooky/store"
	"gopkg.in/mgo.v2/bson"
)

func GetCurentAccount(r *rest.Request) *bson.ObjectId {
	if rv, ok := r.Env["REMOTE_USER"]; ok {
		id := bson.ObjectIdHex(rv.(string))
		return &id
	}
	return nil
}

func GetBase(r *rest.Request) *models.Base {
	if rv, ok := r.Env["MODELS_BASE"]; ok {
		return rv.(*models.Base)
	}
	panic("Missing BaseMiddleware!")
}

type BaseMiddleware struct {
	Store *store.Store
}

func (mw *BaseMiddleware) MiddlewareFunc(next rest.HandlerFunc) rest.HandlerFunc {
	return func(w rest.ResponseWriter, r *rest.Request) {
		db := mw.Store.DB()
		defer db.Session.Close()
		r.Env["MODELS_BASE"] = models.NewBase(db)
		next(w, r)
	}
}

func authenticate(adminPassword string) func(account string, key string, r *rest.Request) bool {
	return func(account string, key string, r *rest.Request) bool {
		if account == "admin" && key == adminPassword {
			return true
		}
		b := GetBase(r)
		if bson.IsObjectIdHex(account) == false {
			fmt.Printf("Authentication error: invalid account %s", account)
			return false
		}
		res, err := b.AuthenticateAccount(bson.ObjectIdHex(account), key)
		if err != nil {
			fmt.Printf("Authentication error: %s\n", err.Error())
			return false
		}
		return res
	}
}

func authorize(adminPassword string) func(account string, r *rest.Request) bool {
	return func(account string, r *rest.Request) bool {
		if account == "admin" {
			return true
		}
		url := r.URL.String()
		if account != "" && url == "/authenticate" {
			return true
		}
		if strings.HasPrefix(url, "/accounts/"+account) == true {
			return true
		}
		return false
	}
}

func Authenticate(w rest.ResponseWriter, r *rest.Request) {
	account := ""
	if ru, ok := r.Env["REMOTE_USER"]; ok == true {
		account = ru.(string)
	}
	w.WriteJson(map[string]string{
		"account": account,
	})
}

// New creates a new instance of the Rest API.
func New(s *store.Store, adminPassword string) (*rest.Api, error) {
	db := s.DB()
	models.NewBase(db).EnsureIndex()
	db.Session.Close()

	api := rest.NewApi()
	api.Use(rest.DefaultDevStack...)
	api.Use(&BaseMiddleware{
		Store: s,
	})
	api.Use(&rest.JsonpMiddleware{
		CallbackNameKey: "cb",
	})
	api.Use(&rest.CorsMiddleware{
		RejectNonCorsRequests: false,
		OriginValidator: func(origin string, request *rest.Request) bool {
			return true
		},
		AllowedMethods:                []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:                []string{"Accept", "Content-Type", "Origin", "Authorization"},
		AccessControlAllowCredentials: true,
		AccessControlMaxAge:           3600,
	})
	authBasic := &AuthBasicMiddleware{
		Realm:         "Hooky",
		Authenticator: authenticate(adminPassword),
		Authorizator:  authorize(adminPassword),
	}
	api.Use(&rest.IfMiddleware{
		Condition: func(r *rest.Request) bool {
			return r.URL.Path != "/status"
		},
		IfTrue: authBasic,
	})
	router, err := rest.MakeRouter(
		rest.Get("/authenticate", Authenticate),
		rest.Post("/accounts", PostAccount),
		rest.Get("/accounts", GetAccounts),
		rest.Get("/accounts/:account", GetAccount),
		rest.Patch("/accounts/:account", PatchAccount),
		rest.Delete("/accounts/:account", DeleteAccount),
		rest.Delete("/accounts/:account/applications", DeleteApplications),
		rest.Get("/accounts/:account/applications", GetApplications),
		rest.Get("/accounts/:account/applications/:application", GetApplication),
		rest.Put("/accounts/:account/applications/:application", PutApplication),
		rest.Delete("/accounts/:account/applications/:application", DeleteApplication),
		rest.Get("/accounts/:account/applications/:application/queues", GetQueues),
		rest.Put("/accounts/:account/applications/:application/queues/:queue", PutQueue),
		rest.Delete("/accounts/:account/applications/:application/queues/:queue", DeleteQueue),
		rest.Get("/accounts/:account/applications/:application/queues/:queue", GetQueue),
		rest.Delete("/accounts/:account/applications/:application/queues", DeleteQueues),
		rest.Post("/accounts/:account/applications/:application/tasks", PutTask),
		rest.Get("/accounts/:account/applications/:application/tasks", GetTasks),
		rest.Delete("/accounts/:account/applications/:application/tasks", DeleteTasks),
		rest.Put("/accounts/:account/applications/:application/tasks/:task", PutTask),
		rest.Get("/accounts/:account/applications/:application/tasks/:task", GetTask),
		rest.Delete("/accounts/:account/applications/:application/tasks/:task", DeleteTask),
		rest.Post("/accounts/:account/applications/:application/tasks/:task/attempts", PostAttempt),
		rest.Get("/accounts/:account/applications/:application/tasks/:task/attempts", GetAttempts),
		rest.Get("/accounts/:account/applications/:application/tasks/:task/attempts/:attempt", GetAttempt),
		rest.Get("/status", GetStatus),
	)
	if err != nil {
		return nil, err
	}
	api.SetApp(router)

	return api, nil
}
