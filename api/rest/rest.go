package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/axpira/backend/entity"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

func NewRestTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

type amountRest struct {
	value int64
}

func NewAmountRest(a int64) *amountRest {
	if a == 0 {
		return nil
	}
	return &amountRest{a}

}

func (a amountRest) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%.02f"`, float64(a.value)*math.Pow10(-2))), nil
}

func (a *amountRest) UnmarshalJSON(data []byte) (err error) {
	v, err := strconv.ParseFloat(string(data[1:len(data)-1]), 10)
	if err != nil {
		return err
	}
	a.value = int64(v * 100)
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

func (e ExpenseRest) ToExpense() entity.Expense {
	exp := entity.Expense{
		Id:    e.Id,
		Where: e.Where,
		Who:   e.Who,
		What:  e.What,
	}
	if e.Amount != nil {
		exp.Amount = e.Amount.value
	}
	if e.When != nil {
		exp.When = e.When.UTC()
	}
	return exp
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
	l := log.Ctx(ctx)
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		l := log.Ctx(r.Context())
		w.Write([]byte("hi"))
		l.Info().Msgf("hello")
	})

	// // RESTy routes for "articles" resource
	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.SetHeader("Content-Type", "application/json"))
		r.Use(TraceID)
		r.Use(LogHandler(l))
		r.Use(middleware.Timeout(60 * time.Second))
		r.NotFound(http.HandlerFunc(notFoundHandler))
		r.Route("/expense", func(r chi.Router) {
			r.Post("/", createExpense(repo))
			r.Route("/{expenseID}", func(r chi.Router) {
				r.Get("/", getExpense(repo))
				r.Delete("/", deleteExpense(repo))
			})
		})
	})
	return r
}

type Error struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewError(code, message string) Error {
	return Error{
		Code:    code,
		Message: message,
	}
}

func (h Error) Error() string {
	return fmt.Sprintf("%s: %s", h.Code, h.Message)
}

type HttpError struct {
	StatusCode    int
	StatusMessage string
	Detail        Error
}

func NewHttpError(code int, msg string, detail Error) HttpError {
	if code == 0 {
		code = http.StatusInternalServerError
	}
	if msg == "" {
		msg = http.StatusText(code)
	}
	return HttpError{
		StatusCode:    code,
		StatusMessage: msg,
		Detail:        detail,
	}
}

func (h HttpError) Error() string {
	return fmt.Sprintf(
		"[%d] %s %q",
		h.StatusCode,
		h.StatusMessage,
		h.Detail.Error(),
	)
}

func fillHttpError(w http.ResponseWriter, err error) bool {
	if err != nil {
		var httpError HttpError
		if errors.As(err, &httpError) {
			w.WriteHeader(httpError.StatusCode)
			err := json.NewEncoder(w).Encode(httpError.Detail)
			if err != nil {
				panic(err)
			}

		}
		return true
	}
	return false
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	fillHttpError(w,
		NewHttpError(http.StatusNotFound, "",
			NewError("URL_NOT_FOUND", fmt.Sprintf("URL '%s' not found", r.URL.String())),
		),
	)
}

func validateError(w http.ResponseWriter, err error) bool {
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			fillHttpError(w, NewHttpError(http.StatusNotFound, "",
				NewError("NOT_FOUND", fmt.Sprintf("expense not found"))),
			)
		} else {
			fillHttpError(w, NewHttpError(0, "",
				NewError("", ""),
			))
		}
		return true
	}
	return false
}

func createExpense(repo ExpenseRepository) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		expense := new(ExpenseRest)
		err := json.NewDecoder(r.Body).Decode(expense)
		if validateError(w, err) {
			log.Ctx(ctx).Err(err).Msg("error on decode")
			return
		}
		id, err := repo.Create(ctx, expense.ToExpense())
		if validateError(w, err) {
			log.Ctx(ctx).Err(err).Msg("error on create")
			return
		}
		w.Write([]byte(`{"id":"` + id + `"}`))
	}
}
func getExpense(repo ExpenseRepository) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		expenseID := chi.URLParam(r, "expenseID")
		expense, err := repo.Get(ctx, expenseID)
		if validateError(w, err) {
			log.Ctx(ctx).Err(err).Msg("error on consult")
			return
		}
		err = json.NewEncoder(w).Encode(NewExpenseRestFromExpense(expense))
		if validateError(w, err) {
			return
		}
	}
}

func deleteExpense(repo ExpenseRepository) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		expenseID := chi.URLParam(r, "expenseID")
		err := repo.Delete(ctx, expenseID)
		if validateError(w, err) {
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
func (s *service) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *service) Status() error {
	return nil
}
