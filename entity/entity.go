package entity

import (
	"fmt"
	"strings"
	"time"
	// "github.com/google/uuid"
)

type Tags struct {
	values []string
}

func NewTags(tags ...string) (*Tags, error) {
	t := new(Tags)
	err := t.Set(tags...)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func MustNewTags(tags ...string) *Tags {
	t, err := NewTags(tags...)
	if err != nil {
		panic(err)
	}
	return t
}

func validateTag(err error, tag string) (string, error) {
	tag = strings.Trim(tag, " ")
	if len(tag) == 0 {
		return "", NewFieldError(err, "tag", "no_empty", "can't be empty")
	}
	if strings.Contains(tag, " ") {
		return "", NewFieldError(err, "tag", "no_space", "can't have spaces")
	}
	return tag, nil
}

func (t *Tags) Contains(tag string) int {
	for i, v := range t.values {
		if v == tag {
			return i
		}
	}
	return -1
}

func (t *Tags) Add(tags ...string) error {
	var err error
	for _, tag := range tags {
		var err1 error
		if tag, err1 = validateTag(err, tag); err1 != nil {
			err = err1
			continue
		}
		if t.Contains(tag) == -1 {
			t.values = append(t.values, tag)
		}
	}
	return err
}

func (t *Tags) Del(tags ...string) {
	for _, tag := range tags {
		if i := t.Contains(tag); i > -1 {
			t.values = append(t.values[:i], t.values[i+1:]...)
		}
	}
}

func (t *Tags) Set(tags ...string) error {
	t.values = make([]string, 0, len(tags))
	return t.Add(tags...)
}

func (t Tags) IsZero() bool {
	return len(t.values) == 0
}

func (t Tags) Value() []string {
	return t.values
}

func (t Tags) String() string {
	return fmt.Sprintf("%v", t.values)
}

type Expense struct {
	Id     string
	Amount int64
	When   time.Time
	Where  string
	Who    string
	What   string
}

type UpdateExpenseFunc func(*Expense) error

type TagAction int

const (
	SET TagAction = iota
	ADD
	DEL
)

func updateTags(tags *Tags, action TagAction, values ...string) error {
	if tags == nil {
		tags = new(Tags)
	}

	switch action {
	case SET:
		return tags.Set(values...)
	case ADD:
		return tags.Add(values...)
	case DEL:
		tags.Del(values...)
	}
	return nil
}

type OpFilterType int

const (
	EQ OpFilterType = iota + 1
	LT
	GT
	LE
	GE
)

type IntFilter struct {
	Type  OpFilterType
	Value int
}

type TimeFilter struct {
	Type  OpFilterType
	Value time.Time
}

type StrFilterType int

const (
	EQUALS StrFilterType = iota + 1
	REGEX
)

type StrFilter struct {
	Type  StrFilterType
	Value string
}

type ExpenseFilter struct {
	Amount []IntFilter
	When   []TimeFilter
	// 	Where         URN
	// 	Who           URN
	What          []StrFilter
	Tag           []StrFilter
	Category      []StrFilter
	PaymentMethod []StrFilter
	// 	// Details       []Detail
	// 	Metadata  map[string]string
	CreatedAt []TimeFilter
	UpdatedAt []TimeFilter
}

func MustNewExpenseFilter(filters ...func(ef *ExpenseFilter) error) *ExpenseFilter {
	f, err := NewExpenseFilter(filters...)
	if err != nil {
		panic(err)
	}
	return f
}

func NewExpenseFilter(filters ...func(ef *ExpenseFilter) error) (*ExpenseFilter, error) {
	f := &ExpenseFilter{}
	for _, filterFunc := range filters {
		err := filterFunc(f)
		if err != nil {
			return nil, err
		}

	}
	return f, nil
}

func FilterAmountInt(t OpFilterType, value int) func(*ExpenseFilter) error {
	return func(e *ExpenseFilter) error {
		e.Amount = append(e.Amount, IntFilter{t, value})
		return nil
	}
}
