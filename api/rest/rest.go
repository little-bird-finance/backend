package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
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

func NewRestTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

type amountRest struct {
	big.Rat
}

func NewAmountRest(a big.Rat) *amountRest {
	if f, _ := a.Float64(); f == 0 {
		return nil
	}
	return &amountRest{a}
}
func (a amountRest) MarshalJSON() ([]byte, error) {
	return []byte(`"` + a.FloatString(2) + `"`), nil
}
func (a *amountRest) UnmarshalJSON(data []byte) (err error) {
	// Fractional seconds are handled implicitly by Parse.
	_, ok := a.SetString(string(data))
	if !ok {
		return errors.New("error on parse amount")
	}
	return nil
}

type ExpenseRest struct {
	Id     string      `json:"id,omitempty"`
	Amount *amountRest `json:"amount,omitempty"`
	When   *time.Time  `json:"when,omitempty"`
	Where  string      `json:"where,omitempty"`
	Who    string      `json:"who,omitempty"`
	What   string      `json:"what,omitempty"`
}

func NewExpenseRestFromExpense(e entity.Expense) ExpenseRest {
	return ExpenseRest{
		Id:     e.Id,
		Amount: NewAmountRest(e.Amount),
		When:   NewRestTime(e.When),
		Where:  e.Where,
		Who:    e.Who,
		What:   e.What,
	}
}

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

func New(ctx context.Context, repo ExpenseRepository) (Service, error) {
	srv := &http.Server{
		Addr:    ":3000",
		Handler: createHandler(ctx, repo),
	}
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

func createHandler(ctx context.Context, repo ExpenseRepository) http.Handler {
	// l := log.Ctx(ctx)
	r := chi.NewRouter()
	// r.Use(middleware.RequestID)
	// r.Use(LogHandler(l))
	// r.Use(middleware.RealIP)
	// r.Use(middleware.Logger)
	// r.Use(middleware.Recoverer)
	// r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		l := log.Ctx(r.Context())
		w.Write([]byte("hi"))
		l.Info().Msgf("hello")
	})

	// // RESTy routes for "articles" resource
	r.Route("/expense", func(r chi.Router) {
		// 	r.With(paginate).Get("/", listArticles)                           // GET /articles
		// 	r.With(paginate).Get("/{month}-{day}-{year}", listArticlesByDate) // GET /articles/01-16-2017

		// 	r.Post("/", createArticle)       // POST /articles
		// 	r.Get("/search", searchArticles) // GET /articles/search

		// 	// Regexp url parameters:
		// 	r.Get("/{articleSlug:[a-z-]+}", getArticleBySlug) // GET /articles/home-is-toronto

		// 	// Subrouters:
		r.Route("/{expenseID}", func(r chi.Router) {
			// 		r.Use(ArticleCtx)
			r.Get("/", getExpenseHandler(repo)) // GET /articles/123
			// 		r.Put("/", updateArticle)    // PUT /articles/123
			// 		r.Delete("/", deleteArticle) // DELETE /articles/123
		})
	})
	return r
}

func getExpenseHandler(repo ExpenseRepository) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		expenseID := chi.URLParam(r, "expenseID")
		expense, err := repo.Get(ctx, expenseID)
		if err != nil {
			fmt.Fprintf(w, "%v", err)
			// w.Write([]byte(fmt.err))
			// http.Error(w, http.StatusText(422), 422)

			return
		}
		err = json.NewEncoder(w).Encode(NewExpenseRestFromExpense(expense))
		if err != nil {
			fmt.Fprintf(w, "%v", err)
			// http.Error(w, http.StatusText(422), 422)
			return
		}
	}
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
