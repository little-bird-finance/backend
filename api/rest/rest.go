package rest

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/axpira/backend/entity"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// import (
// 	"encoding/json"
// 	"github.com/axpira/backend/entity"
// 	"google.golang.org/protobuf/encoding/protojson"
// 	"log"
// )

// func main() {
// 	log.Printf("Teste")
// 	id, _ := entity.NewULID(0, entity.DefaultEntropy())
// 	log.Printf("%v", id)

// 	j, _ := protojson.Marshal(id)
// 	log.Printf("%v", string(j))
// 	j, _ = json.Marshal(id)
// 	log.Printf("%v", string(j))

// 	tmp := entity.ULID{}
// 	j, _ = protojson.Marshal(&tmp)
// 	log.Printf("%v", string(j))
// 	j, _ = json.Marshal(&tmp)
// 	log.Printf("%v", string(j))

// }

type Service interface {
	Start(context.Context)
	Stop(context.Context) error
	Status() error
}

type ExpenseRepository interface {
	Create(ctx context.Context, expense entity.Expense) (string, error)
	Update(ctx context.Context, expense entity.Expense) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (entity.Expense, error)
	Search(context.Context, *entity.ExpenseFilter) ([]entity.Expense, error)
}

type service struct {
	srv *http.Server
	wg  *sync.WaitGroup
}

func LogHandler(logger *zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			l := logger.
				With().
				Str("request-id", middleware.GetReqID(ctx)).
				Logger()
			ctx = l.WithContext(ctx)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

func New(repo ExpenseRepository, ctx context.Context) (Service, error) {
	l := log.Ctx(ctx)
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(LogHandler(l))
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		l := log.Ctx(r.Context())
		w.Write([]byte("hi"))
		l.Info().Msgf("hello")
	})

	// // RESTy routes for "articles" resource
	// r.Route("/expense", func(r chi.Router) {
	// 	r.With(paginate).Get("/", listArticles)                           // GET /articles
	// 	r.With(paginate).Get("/{month}-{day}-{year}", listArticlesByDate) // GET /articles/01-16-2017

	// 	r.Post("/", createArticle)       // POST /articles
	// 	r.Get("/search", searchArticles) // GET /articles/search

	// 	// Regexp url parameters:
	// 	r.Get("/{articleSlug:[a-z-]+}", getArticleBySlug) // GET /articles/home-is-toronto

	// 	// Subrouters:
	// 	r.Route("/{articleID}", func(r chi.Router) {
	// 		r.Use(ArticleCtx)
	// 		r.Get("/", getArticle)       // GET /articles/123
	// 		r.Put("/", updateArticle)    // PUT /articles/123
	// 		r.Delete("/", deleteArticle) // DELETE /articles/123
	// 	})
	// })
	srv := &http.Server{Addr: ":3000", Handler: r}
	return &service{
		srv: srv,
	}, nil
}

func (s *service) Start(ctx context.Context) {
	go func() {
		if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Ctx(ctx).Fatal().Msgf("error on start rest service: %v", err)
		}
	}()
}

func (s *service) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *service) Status() error {
	return nil
}

// func getExpense(w http.ResponseWriter, r *http.Request) {
//   ctx := r.Context()
//   article, ok := ctx.Value("article").(*Article)
//   if !ok {
//     http.Error(w, http.StatusText(422), 422)
//     return
//   }
//   w.Write([]byte(fmt.Sprintf("title:%s", article.Title)))
// }
