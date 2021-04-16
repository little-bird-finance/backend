package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/axpira/backend/entity"
	"github.com/oklog/ulid/v2"
)

var (
	ErrUnknown           = errors.New("uknown error")
	expenseRowColumnsArr = []string{
		"amount",
		"when",
		"where",
		"who",
		"what",
	}
	expenseRowColumns = strings.Join(expenseRowColumnsArr, ",")
)

type ExpenseRow struct {
	Id     string
	Amount bigRatAdapter
	When   time.Time
	Where  string
	Who    string
	What   string
}

func NewExpenseRowFromExpense(e entity.Expense) ExpenseRow {
	return ExpenseRow{
		Id:     e.Id,
		Amount: bigRatAdapter{e.Amount},
		When:   e.When,
		Where:  e.Where,
		Who:    e.Who,
		What:   e.What,
	}
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
	expense.Amount = e.Amount.Rat
	expense.When = e.When
	expense.Where = e.Where
	expense.Who = e.Who
	expense.What = e.What
	return expense, nil
}

func (e ExpenseRow) NamedArgs() []sql.NamedArg {
	var args []sql.NamedArg
	if e.Amount.Cmp(&big.Rat{}) != 0 {
		args = append(args, sql.Named("amount", e.Amount))
	}
	if !e.When.IsZero() {
		args = append(args, sql.Named("when", e.When))
	}
	if e.Where != "" {
		args = append(args, sql.Named("where", e.Where))
	}
	if e.Who != "" {
		args = append(args, sql.Named("who", e.Who))
	}
	if e.What != "" {
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

type expenseRepository struct {
	db      *sql.DB
	entropy io.Reader
}

func NewExpenseRepository(db *sql.DB) Repository {

	return expenseRepository{
		db:      db,
		entropy: ulid.Monotonic(rand.New(rand.NewSource(time.Now().Local().UnixNano())), 0),
	}
}

type bigRatAdapter struct {
	big.Rat
}

func (b bigRatAdapter) Value() (driver.Value, error) {
	return b.String(), nil
}

func (b *bigRatAdapter) Scan(value interface{}) error {
	return nil
}

func (r expenseRepository) Create(ctx context.Context, expense entity.Expense) (string, error) {
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
		valueStr.WriteString(", @" + namedArg.Name)
		args[i] = namedArg
	}

	query := fmt.Sprintf("INSERT INTO expense (%s) VALUES (%s);", keyStr.String()[1:], valueStr.String()[1:])
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
		sql.Named("id", expense.Id),
		sql.Named("updatedAt", time.Now().UTC()),
	)
	var fieldsStr strings.Builder
	args := make([]interface{}, len(namedArgs))
	for i, namedArg := range namedArgs {
		fieldsStr.WriteString(fmt.Sprintf(", %s = @%s ", namedArg.Name, namedArg.Name))
		args[i] = namedArg
	}

	query := fmt.Sprintf("UPDATE expense SET%sWHERE id = @id;", fieldsStr.String()[1:])
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
		"DELETE FROM expense WHERE id = @id;",
		sql.Named("id", id),
	)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnknown, err)
	}
	return nil
}

func (r expenseRepository) Get(ctx context.Context, id string) (entity.Expense, error) {
	if strings.TrimSpace(id) == "" {
		return entity.Expense{}, entity.NewFieldError(nil, "id", "empty", "can't be empty")
	}
	row := ExpenseRow{
		Id: id,
	}
	err := r.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT %s FROM expense WHERE id = @id;", expenseRowColumns),
		sql.Named("id", id),
	).Scan(row.Scan()...)
	if err != nil {
		return entity.Expense{}, err
	}
	return row.ToExpense()
}

func (r expenseRepository) Search(context.Context, *entity.ExpenseFilter) ([]entity.Expense, error) {
	return nil, nil
}
