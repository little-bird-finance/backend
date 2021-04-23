package rest

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/axpira/backend/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// func Test(t *testing.T) {
// 	handler := func(w http.ResponseWriter, r *http.Request) {
// 		io.WriteString(w, "<html><body>Hello World!</body></html>")
// 	}

// 	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
// 	w := httptest.NewRecorder()
// 	handler(w, req)

// 	resp := w.Result()
// 	body, _ := io.ReadAll(resp.Body)

// 	fmt.Println(resp.StatusCode)
// 	fmt.Println(resp.Header.Get("Content-Type"))
// 	fmt.Println(string(body))
// }

// func Test2(t *testing.T) {
// 	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		fmt.Fprintln(w, "Hello, client")
// 	}))
// 	defer ts.Close()

// 	res, err := http.Get(ts.URL)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	greeting, err := io.ReadAll(res.Body)
// 	res.Body.Close()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	fmt.Printf("%s", greeting)
// }

type mockExpenseRepo struct {
	mock.Mock
}

func (m *mockExpenseRepo) Create(ctx context.Context, expense entity.Expense) (string, error) {
	args := m.Called(ctx, expense)
	return args.String(0), args.Error(1)
}
func (m *mockExpenseRepo) Update(ctx context.Context, expense entity.Expense) error {
	args := m.Called(ctx, expense)
	return args.Error(0)
}
func (m *mockExpenseRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *mockExpenseRepo) Get(ctx context.Context, id string) (entity.Expense, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(entity.Expense), args.Error(1)
}

func (m *mockExpenseRepo) Search(context.Context, *entity.ExpenseFilter) ([]entity.Expense, error) {
	return nil, nil
}

func TestGetExpense(t *testing.T) {

	now := time.Now()
	tests := map[string]struct {
		id           string
		mockExpense  entity.Expense
		mockErr      error
		handlerError bool
		wantResult   []byte
		wantStatus   int
	}{
		"success": {
			id: "1",
			mockExpense: entity.Expense{
				Id:     "1",
				Amount: 120,
				What:   "my what",
				When:   now,
				Where:  "my where",
				Who:    "my who",
			},
			wantStatus: 200,
			wantResult: []byte(`{
				"id":     "1",
				"amount": "1.20",
				"what":   "my what",
				"when": "` + now.Format(time.RFC3339Nano) + `",
				"where": "my where",
				"who": "my who"
			}`),
		},
		"success amount": {
			id: "2",
			mockExpense: entity.Expense{
				Id:     "2",
				Amount: 230,
			},
			wantStatus: 200,
			wantResult: []byte(`{
				"id":     "2",
				"amount": "2.30"
			}`),
		},
		"success id": {
			id: "3",
			mockExpense: entity.Expense{
				Id: "3",
			},
			wantStatus: 200,
			wantResult: []byte(`{
				"id":     "3"
			}`),
		},
		"expense not found": {
			id:         "4",
			wantStatus: 404,
			wantResult: []byte(`{
				"code":     "NOT_FOUND",
				"message": "expense not found"
			}`),
			mockErr: entity.ErrNotFound,
		},
		"unknown error": {
			id:         "5",
			wantStatus: 500,
			wantResult: []byte("{}"),
			mockErr:    errors.New("unknown error"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mockedExpense := new(mockExpenseRepo)
			mockedExpense.
				On("Get", mock.Anything, tc.id).
				Return(tc.mockExpense, tc.mockErr)
			ctx := context.Background()
			ts := httptest.NewServer(createHandler(ctx, mockedExpense))
			defer ts.Close()

			res, err := http.Get(ts.URL + "/api/expense/" + tc.id)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.wantStatus, res.StatusCode)

			got, err := io.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, res.Header.Get("content-type"), "application/json")

			if !tc.handlerError {
				mockedExpense.AssertExpectations(t)
			}
			assert.JSONEq(t, string(tc.wantResult), string(got))

		})
	}
}
