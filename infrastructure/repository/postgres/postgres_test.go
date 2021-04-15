package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/axpira/backend/entity"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

	// "github.com/stretchr/testify/mock"
	"github.com/DATA-DOG/go-sqlmock"
)

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

type timeMatch struct {
	expected time.Time
}

func (a timeMatch) Match(v driver.Value) bool {
	if t, ok := v.(time.Time); ok {
		if t.Location() != a.expected.Location() {
			return false
		}
		if math.Abs(t.Sub(a.expected).Seconds()) < 5 {
			return true
		}
	}
	return false
}

type anyULID struct{}

func (a anyULID) Match(v driver.Value) bool {
	if str, ok := v.(string); ok {
		if _, err := ulid.Parse(str); err == nil {
			return true
		}
	}
	return false
}

func newRandomExpense() entity.Expense {
	return entity.Expense{
		Id:     String(36),
		Amount: *big.NewRat(12, 34),
		When:   time.Now().Add(-time.Duration(seededRand.Intn(3600)) * time.Second).UTC(),
		Where:  String(5),
		Who:    String(5),
		What:   String(15),
	}
}

func TestCreate(t *testing.T) {
	repo := ExpenseRepository{entropy: bytes.NewReader([]byte(""))}
	id, err := repo.Create(context.Background(), entity.Expense{})
	assert.Error(t, err, "must return error on invalid ULID")
	assert.Empty(t, id)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	repo = NewExpenseRepository(db)

	tests := map[string]struct {
		expense entity.Expense
		args    []driver.Value
		wantErr error
		mockErr error
	}{
		"must execute the insert query with all named args": {
			expense: newRandomExpense(),
		},
		"must execute the insert query with just field sent": {
			expense: entity.Expense{
				Amount: *big.NewRat(1, 2),
			},
			args: []driver.Value{
				sql.Named("amount", bigRatAdapter{*big.NewRat(1, 2)}),
				anyULID{},
				timeMatch{time.Now().UTC()},
				timeMatch{time.Now().UTC()},
			},
		},
		"must return error on database error ": {
			expense: newRandomExpense(),
			wantErr: ErrUnknown,
			mockErr: errors.New(String(10)),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			args := tc.args
			if args == nil {
				args = append(args, sql.Named("amount", bigRatAdapter{tc.expense.Amount}))
				args = append(args, sql.Named("when", tc.expense.When))
				args = append(args, sql.Named("where", tc.expense.Where))
				args = append(args, sql.Named("who", tc.expense.Who))
				args = append(args, sql.Named("what", tc.expense.What))
				args = append(args, anyULID{})
				args = append(args, timeMatch{time.Now().UTC()})
				args = append(args, timeMatch{time.Now().UTC()})
			}
			mock.
				ExpectExec("INSERT INTO expense (.*) VALUES (.+);").
				WithArgs(args...).
				WillReturnError(tc.mockErr).
				WillReturnResult(sqlmock.NewResult(1, 1))

			_, gotErr := repo.Create(context.Background(), tc.expense)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectation error: %s", err)
			}

			if gotErr != nil {
				if !errors.As(gotErr, &tc.wantErr) {
					t.Errorf("want error %q and got %q", tc.wantErr, gotErr)
				}
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error %q was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	repo := NewExpenseRepository(db)

	err = repo.Update(context.Background(), entity.Expense{})
	if err == nil {
		t.Errorf("must return error on empty id")
	}
	tests := map[string]struct {
		// methods   []string
		// wantQuery string
		expense entity.Expense
		args    []driver.Value
		wantErr error
		mockErr error
	}{
		"must execute the update query with all named args": {
			expense: newRandomExpense(),
		},
		"must execute the update query with just field sent": {
			expense: entity.Expense{
				Id:     "123456",
				Amount: *big.NewRat(1, 2),
			},
			args: []driver.Value{
				sql.Named("amount", bigRatAdapter{*big.NewRat(1, 2)}),
				sql.Named("id", "123456"),
				timeMatch{time.Now().UTC()},
			},
		},
		"must return error on database error ": {
			expense: newRandomExpense(),
			wantErr: ErrUnknown,
			mockErr: errors.New(String(10)),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			args := tc.args
			if args == nil {
				args = append(args, sql.Named("amount", bigRatAdapter{tc.expense.Amount}))
				args = append(args, sql.Named("when", tc.expense.When))
				args = append(args, sql.Named("where", tc.expense.Where))
				args = append(args, sql.Named("who", tc.expense.Who))
				args = append(args, sql.Named("what", tc.expense.What))
				args = append(args, sql.Named("id", tc.expense.Id))
				args = append(args, timeMatch{time.Now().UTC()})
			}
			mock.
				ExpectExec("UPDATE expense SET (.+) WHERE id = @id;").
				WithArgs(args...).
				WillReturnError(tc.mockErr).
				WillReturnResult(sqlmock.NewResult(1, 1))

			gotErr := repo.Update(context.Background(), tc.expense)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectation error: %s", err)
			}

			if gotErr != nil {
				if !errors.As(gotErr, &tc.wantErr) {
					t.Errorf("want error %q and got %q", tc.wantErr, gotErr)
				}
			}
		})
	}
}

func TestDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	repo := NewExpenseRepository(db)

	err = repo.Delete(context.Background(), "")
	if err == nil {
		t.Errorf("must return error on empty id")
	}

	tests := map[string]struct {
		id      string
		wantErr bool
		mockErr error
	}{
		"must execute the delete": {
			id: String(36),
		},
		"must return error on database error ": {
			id:      String(36),
			wantErr: true,
			mockErr: errors.New(String(10)),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock.
				ExpectExec("DELETE FROM expense WHERE id = @id").
				WithArgs(tc.id).
				WillReturnError(tc.mockErr).
				WillReturnResult(sqlmock.NewResult(1, 1))

			gotErr := repo.Delete(context.Background(), tc.id)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectation error: %s", err)
			}

			if tc.wantErr {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}
		})
	}
}

func TestGet(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	repo := NewExpenseRepository(db)

	_, err = repo.Get(context.Background(), "")
	if err == nil {
		t.Errorf("must return error on empty id")
	}

	tests := map[string]struct {
		id          string
		wantErr     error
		mockErr     error
		columns     []string
		wantExpense entity.Expense
	}{
		"must execute the query": {
			columns:     []string{"amount", "when", "where", "who", "what"},
			id:          String(36),
			wantExpense: newRandomExpense(),
		},
		"must return error on database error ": {
			columns: []string{"amount", "when", "where", "who", "what"},
			id:      String(36),
			wantErr: ErrUnknown,
			mockErr: errors.New(String(10)),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock.
				ExpectQuery(
					fmt.Sprintf(
						"SELECT %v FROM expense WHERE id = @id",
						strings.Join(tc.columns, ","),
					),
				).
				WithArgs(sql.Named("id", tc.id)).
				WillReturnRows(
					sqlmock.NewRows(tc.columns).AddRow(
						bigRatAdapter{tc.wantExpense.Amount},
						tc.wantExpense.When,
						tc.wantExpense.Where,
						tc.wantExpense.Who,
						tc.wantExpense.What,
					),
				).
				WillReturnError(tc.mockErr)

			gotExpense, gotErr := repo.Get(context.Background(), tc.id)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectation error: %s", err)
			}

			if gotErr != nil {
				if !errors.As(gotErr, &tc.wantErr) {
					t.Errorf("want error %q and got %q", tc.wantErr, gotErr)
				}
			} else {
				tc.wantExpense.Id = tc.id
			}

			if diff := cmp.Diff(tc.wantExpense, gotExpense, cmpopts.IgnoreUnexported(big.Rat{}), cmpopts.IgnoreUnexported(entity.Tags{})); diff != "" {
				t.Errorf("Expense mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func String(length int) string {
	return StringWithCharset(length, charset)
}
