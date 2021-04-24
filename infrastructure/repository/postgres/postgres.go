package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"

	"github.com/axpira/backend/entity"
	"github.com/axpira/backend/entity/config"

	_ "github.com/jackc/pgx/v4/stdlib"

	// _ "github.com/lib/pq"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog/log"
)

const TABLE_NAME = "tb_expense"

var (
	ErrUnknown           = errors.New("uknown error")
	expenseRowColumnsArr = []string{
		"amount",
		"timestamp",
		"place",
		"who",
		"what",
	}
	expenseRowColumns = strings.Join(expenseRowColumnsArr, ",")
)

type ExpenseRow struct {
	Id     string
	Amount sql.NullInt64
	When   sql.NullTime
	Where  sql.NullString
	Who    sql.NullString
	What   sql.NullString
}

func NewExpenseRowFromExpense(e entity.Expense) ExpenseRow {
	row := ExpenseRow{
		Id: e.Id,
	}
	if e.Amount > 0 {
		row.Amount = sql.NullInt64{Int64: e.Amount, Valid: true}
	}
	if !e.When.IsZero() {
		row.When = sql.NullTime{Time: e.When, Valid: true}
	}
	if e.Where != "" {
		row.Where = sql.NullString{String: e.Where, Valid: true}
	}
	if e.Who != "" {
		row.Who = sql.NullString{String: e.Who, Valid: true}
	}
	if e.What != "" {
		row.What = sql.NullString{String: e.What, Valid: true}
	}
	return row
}

func (e *ExpenseRow) Scan() []interface{} {
	return []interface{}{
		&e.Amount,
		&e.When,
		&e.Where,
		&e.Who,
		&e.What,
	}
}

func (e ExpenseRow) ToExpense() (entity.Expense, error) {
	expense := entity.Expense{}
	expense.Id = e.Id
	expense.Amount = e.Amount.Int64
	expense.When = e.When.Time.UTC()
	expense.Where = e.Where.String
	expense.Who = e.Who.String
	expense.What = e.What.String
	return expense, nil
}

func (e ExpenseRow) NamedArgs() []sql.NamedArg {
	var args []sql.NamedArg
	if e.Amount.Valid {
		args = append(args, sql.Named("amount", e.Amount))
	}
	if e.When.Valid {
		args = append(args, sql.Named("timestamp", e.When))
	}
	if e.Where.Valid {
		args = append(args, sql.Named("place", e.Where))
	}
	if e.Who.Valid {
		args = append(args, sql.Named("who", e.Who))
	}
	if e.What.Valid {
		args = append(args, sql.Named("what", e.What))
	}
	return args
}

type Repository interface {
	Create(ctx context.Context, expense entity.Expense) (string, error)
	Update(ctx context.Context, expense entity.Expense) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (entity.Expense, error)
	Search(context.Context, *entity.ExpenseFilter) ([]entity.Expense, error)
}

type DB interface {
	ExecContext(_ context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

type expenseRepository struct {
	db      DB
	entropy io.Reader
}

func NewExpenseRepository() (Repository, error) {
	db, err := sql.Open("pgx", config.Config.DatabaseUrl)
	if err != nil {
		return nil, err
	}
	return expenseRepository{
		db:      db,
		entropy: defaultEntropy(),
	}, nil
}

func defaultEntropy() io.Reader {
	return ulid.Monotonic(rand.New(rand.NewSource(time.Now().Local().UnixNano())), 0)
}

func (r expenseRepository) Create(ctx context.Context, expense entity.Expense) (string, error) {
	l := log.Ctx(ctx)
	id, err := ulid.New(ulid.Timestamp(time.Now().UTC()), r.entropy)
	if err != nil {
		return "", err
	}
	expense.Id = id.String()
	row := NewExpenseRowFromExpense(expense)
	now := time.Now().UTC()
	namedArgs := append(
		row.NamedArgs(),
		sql.Named("id", expense.Id),
		sql.Named("createdAt", now),
		sql.Named("updatedAt", now),
	)
	var keyStr strings.Builder
	var valueStr strings.Builder
	args := make([]interface{}, len(namedArgs))
	for i, namedArg := range namedArgs {
		keyStr.WriteString("," + namedArg.Name)
		// valueStr.WriteString(", @" + namedArg.Name)
		valueStr.WriteString(fmt.Sprintf(", $%d", i+1))
		args[i] = namedArg
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);", TABLE_NAME, keyStr.String()[1:], valueStr.String()[1:])
	if e := l.Debug(); e.Enabled() {
		for i, a := range args {
			n := a.(sql.NamedArg)
			e = e.Str(fmt.Sprintf("param_%d_%s", i+1, n.Name), fmt.Sprintf("%v", n.Value))
		}
		e.Msgf("runing: %v", query)
	}
	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnknown, err)
	}
	return expense.Id, nil
}

func (r expenseRepository) Update(ctx context.Context, expense entity.Expense) error {
	if strings.TrimSpace(expense.Id) == "" {
		return entity.NewFieldError(nil, "id", "empty", "can't be empty")
	}
	row := NewExpenseRowFromExpense(expense)
	namedArgs := append(
		row.NamedArgs(),
		sql.Named("updatedAt", time.Now().UTC()),
	)
	var fieldsStr strings.Builder
	args := make([]interface{}, len(namedArgs)+1)
	args[0] = sql.Named("id", expense.Id)
	for i, namedArg := range namedArgs {
		fieldsStr.WriteString(fmt.Sprintf(", %s = $%d ", namedArg.Name, i+2))
		args[i+1] = namedArg
	}

	query := fmt.Sprintf("UPDATE %s SET%sWHERE id = $1;", TABLE_NAME, fieldsStr.String()[1:])
	l := log.Ctx(ctx)
	if e := l.Debug(); e.Enabled() {
		for i, a := range args {
			n := a.(sql.NamedArg)
			e = e.Str(fmt.Sprintf("param_%d_%s", i+1, n.Name), fmt.Sprintf("%v", n.Value))
		}
		e.Msgf("runing: %v", query)
	}
	// l.Debug().Msgf("%+v", args)
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnknown, err)
	}
	return nil
}

func (r expenseRepository) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return entity.NewFieldError(nil, "id", "empty", "can't be empty")
	}
	_, err := r.db.ExecContext(
		ctx,
		fmt.Sprintf("DELETE FROM %s WHERE id = $1;", TABLE_NAME),
		sql.Named("id", id),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("id %s was %w", id, entity.ErrNotFound)
		}
		return fmt.Errorf("%w: %v", entity.ErrUnknown, err)
	}
	return nil
}

func (r expenseRepository) Get(ctx context.Context, id string) (entity.Expense, error) {
	l := log.Ctx(ctx)
	l.Info().Msgf("get %s", id)
	if strings.TrimSpace(id) == "" {
		return entity.Expense{}, entity.NewFieldError(nil, "id", "empty", "can't be empty")
	}
	row := ExpenseRow{
		Id: id,
	}
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = $1;", expenseRowColumns, TABLE_NAME)
	l.Debug().Msgf("executing: %s", query)
	if e := l.Debug(); e.Enabled() {
		e.Str("param_1_id", fmt.Sprintf("%v", sql.Named("id", id))).Msgf("runing: %v", query)
	}
	err := r.db.QueryRowContext(ctx,
		query,
		sql.Named("id", id),
	).Scan(row.Scan()...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Expense{}, fmt.Errorf("id %s was %w", id, entity.ErrNotFound)
		}
		return entity.Expense{}, fmt.Errorf("%w: %v", entity.ErrUnknown, err)
	}
	l.Debug().Msgf("%v", row)
	return row.ToExpense()
}

func (r expenseRepository) Search(context.Context, *entity.ExpenseFilter) ([]entity.Expense, error) {
	return nil, nil
}
