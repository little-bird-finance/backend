package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"math"
	"math/big"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/axpira/backend/entity"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
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
		Amount: 1234,
		When:   time.Now().Add(-time.Duration(seededRand.Intn(3600)) * time.Second).UTC(),
		Where:  String(5),
		Who:    String(5),
		What:   String(15),
	}
}

func TestCreate(t *testing.T) {
	l := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().
		Timestamp().
		Caller().
		Str("service", "backend").
		Logger()
	ctx := l.WithContext(context.Background())
	repoErr := expenseRepository{entropy: bytes.NewReader([]byte(""))}
	id, err := repoErr.Create(context.Background(), entity.Expense{})
	assert.Error(t, err, "must return error on invalid ULID")
	assert.Empty(t, id)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	tests := map[string]struct {
		expense   entity.Expense
		args      []driver.Value
		wantQuery string
		wantErr   error
		mockErr   error
	}{
		"must execute the insert query with all named args": {
			expense:   newRandomExpense(),
			wantQuery: "INSERT INTO tb_expense \\(amount,timestamp,place,who,what,id,createdAt,updatedAt\\) VALUES \\( \\$1, \\$2, \\$3, \\$4, \\$5, \\$6, \\$7, \\$8\\);",
		},
		"must execute the insert query with just field sent": {
			expense: entity.Expense{
				Amount: 120,
			},
			wantQuery: "INSERT INTO tb_expense \\(amount,id,createdAt,updatedAt\\) VALUES \\( \\$1, \\$2, \\$3, \\$4\\);",
			args: []driver.Value{
				sql.Named("amount", sql.NullInt64{120, true}),
				anyULID{},
				timeMatch{time.Now().UTC()},
				timeMatch{time.Now().UTC()},
			},
		},
		"must return error on database error ": {
			expense:   newRandomExpense(),
			wantQuery: "INSERT INTO tb_expense \\(amount,timestamp,place,who,what,id,createdAt,updatedAt\\) VALUES \\( \\$1, \\$2, \\$3, \\$4, \\$5, \\$6, \\$7, \\$8\\);",
			wantErr:   ErrUnknown,
			mockErr:   errors.New(String(10)),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			args := tc.args
			if args == nil {
				args = append(args, sql.Named("amount", sql.NullInt64{tc.expense.Amount, true}))
				args = append(args, sql.Named("timestamp", sql.NullTime{tc.expense.When, true}))
				args = append(args, sql.Named("place", sql.NullString{tc.expense.Where, true}))
				args = append(args, sql.Named("who", sql.NullString{tc.expense.Who, true}))
				args = append(args, sql.Named("what", sql.NullString{tc.expense.What, true}))
				args = append(args, anyULID{})
				args = append(args, timeMatch{time.Now().UTC()})
				args = append(args, timeMatch{time.Now().UTC()})
			}
			mock.
				ExpectExec(tc.wantQuery).
				WithArgs(args...).
				WillReturnError(tc.mockErr).
				WillReturnResult(sqlmock.NewResult(1, 1))
			repo := expenseRepository{
				db:      db,
				entropy: defaultEntropy(),
			}

			_, gotErr := repo.Create(ctx, tc.expense)
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
	l := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().
		Timestamp().
		Caller().
		Str("service", "backend-test").
		Logger()
	ctx := l.WithContext(context.Background())
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error %q was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	repo := expenseRepository{
		db:      db,
		entropy: defaultEntropy(),
	}

	err = repo.Update(ctx, entity.Expense{})
	if err == nil {
		t.Errorf("must return error on empty id")
	}
	tests := map[string]struct {
		expense   entity.Expense
		wantQuery string
		args      []driver.Value
		wantErr   error
		mockErr   error
	}{
		"must execute the update query with all named args": {
			expense:   newRandomExpense(),
			wantQuery: "UPDATE tb_expense SET amount = \\$2 , timestamp = \\$3 , place = \\$4 , who = \\$5 , what = \\$6 , updatedAt = \\$7 WHERE id = \\$1;",
		},
		"must execute the update query with just field sent": {
			expense: entity.Expense{
				Id:     "123456",
				Amount: 120,
			},
			wantQuery: "UPDATE tb_expense SET amount = \\$2 , updatedAt = \\$3 WHERE id = \\$1;",
			args: []driver.Value{
				sql.Named("id", "123456"),
				sql.Named("amount", sql.NullInt64{120, true}),
				timeMatch{time.Now().UTC()},
			},
		},
		"must return error on database error ": {
			expense:   newRandomExpense(),
			wantQuery: "UPDATE tb_expense SET amount = \\$2 , timestamp = \\$3 , place = \\$4 , who = \\$5 , what = \\$6 , updatedAt = \\$7 WHERE id = \\$1;",
			wantErr:   ErrUnknown,
			mockErr:   errors.New(String(10)),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			args := tc.args
			if args == nil {
				args = append(args, sql.Named("id", tc.expense.Id))
				args = append(args, sql.Named("amount", sql.NullInt64{tc.expense.Amount, true}))
				args = append(args, sql.Named("timestamp", sql.NullTime{tc.expense.When, true}))
				args = append(args, sql.Named("place", sql.NullString{tc.expense.Where, true}))
				args = append(args, sql.Named("who", sql.NullString{tc.expense.Who, true}))
				args = append(args, sql.Named("what", sql.NullString{tc.expense.What, true}))
				args = append(args, timeMatch{time.Now().UTC()})
			}
			mock.
				ExpectExec(tc.wantQuery).
				WithArgs(args...).
				WillReturnError(tc.mockErr).
				WillReturnResult(sqlmock.NewResult(1, 1))
			repo.Update(ctx, tc.expense)

			gotErr := repo.Update(ctx, tc.expense)
			// db.AssertExpectations(t)
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
	l := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().
		Timestamp().
		Caller().
		Str("service", "backend-test").
		Logger()
	ctx := l.WithContext(context.Background())
	repo := expenseRepository{
		db:      db,
		entropy: defaultEntropy(),
	}

	err = repo.Delete(ctx, "")
	if err == nil {
		t.Errorf("must return error on empty id")
	}

	tests := map[string]struct {
		id        string
		wantQuery string
		wantErr   error
		mockErr   error
	}{
		"must execute the delete": {
			id:        String(36),
			wantQuery: "DELETE FROM tb_expense WHERE id = \\$1;",
		},
		"must return error on database error ": {
			id:        String(36),
			wantQuery: "DELETE FROM tb_expense WHERE id = \\$1;",
			wantErr:   entity.ErrUnknown,
			mockErr:   errors.New(String(10)),
		},
		"must return not found error on database NoRows": {
			id:        String(36),
			wantQuery: "DELETE FROM tb_expense WHERE id = \\$1;",
			wantErr:   entity.ErrNotFound,
			mockErr:   sql.ErrNoRows,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock.
				ExpectExec(tc.wantQuery).
				WithArgs(tc.id).
				WillReturnError(tc.mockErr).
				WillReturnResult(sqlmock.NewResult(1, 1))
			gotErr := repo.Delete(ctx, tc.id)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectation error: %s", err)
			}
			if tc.wantErr != nil {
				assert.ErrorIs(t, gotErr, tc.wantErr)
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
	l := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().
		Timestamp().
		Caller().
		Str("service", "backend-test").
		Logger()
	ctx := l.WithContext(context.Background())
	// db := new(mockDB)
	repo := expenseRepository{
		db:      db,
		entropy: defaultEntropy(),
	}

	_, err = repo.Get(ctx, "")
	if err == nil {
		t.Errorf("must return error on empty id")
	}

	tests := map[string]struct {
		id          string
		wantQuery   string
		wantErr     error
		mockErr     error
		columns     []string
		wantExpense entity.Expense
	}{
		"must execute the query": {
			wantQuery:   "SELECT amount,timestamp,place,who,what FROM tb_expense WHERE id = \\$1;",
			columns:     []string{"amount", "when", "where", "who", "what"},
			id:          String(36),
			wantExpense: newRandomExpense(),
		},
		"must return error on database error ": {
			columns: []string{"amount", "when", "where", "who", "what"},
			id:      String(36),
			wantErr: entity.ErrUnknown,
			mockErr: errors.New(String(10)),
		},
		"must return error on not found": {
			columns: []string{"amount", "when", "where", "who", "what"},
			id:      String(36),
			wantErr: entity.ErrNotFound,
			mockErr: sql.ErrNoRows,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mock.
				ExpectQuery(tc.wantQuery).
				WithArgs(sql.Named("id", tc.id)).
				WillReturnRows(
					sqlmock.NewRows(tc.columns).AddRow(
						tc.wantExpense.Amount,
						tc.wantExpense.When,
						tc.wantExpense.Where,
						tc.wantExpense.Who,
						tc.wantExpense.What,
					),
				).
				WillReturnError(tc.mockErr)

			repo := expenseRepository{
				db:      db,
				entropy: defaultEntropy(),
			}
			gotExpense, gotErr := repo.Get(ctx, tc.id)

			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectation error: %s", err)
			}

			if gotErr != nil {
				assert.ErrorIs(t, gotErr, tc.wantErr)
			} else {
				tc.wantExpense.Id = tc.id
				assert.NoError(t, gotErr)
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
