package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewDB(t *testing.T) {
	a := assert.New(t)
	db := NewDB(logrus.New())
	a.IsType([]*ProduceItem{}, db.Produce)
}

func Test_CodeIsValid(t *testing.T) {
	type args struct {
		c      string
		logger *logrus.Logger
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "oneUpper",
			args: args{c: "A12T-4GH7-QPL9-3N4M", logger: logrus.New()},
			want: true,
		},
		{
			name: "oneLower",
			args: args{c: "a12t-4gh7-qpl9-3n4m", logger: logrus.New()},
			want: true,
		},
		{
			name: "twoUpper",
			args: args{c: "YRT6-72AS-K736-L4AR", logger: logrus.New()},
			want: true,
		},
		{
			name: "twoLower",
			args: args{c: "yrt6-72as-k736-l4ar", logger: logrus.New()},
			want: true,
		},
		{
			name: "threeUpper",
			args: args{c: "TQ4C-VV6T-75ZX-1RMR", logger: logrus.New()},
			want: true,
		},
		{
			name: "threeLower",
			args: args{c: "tq4c-vv6t-75zx-1rmr", logger: logrus.New()},
			want: true,
		},
		{
			name: "fourUpper",
			args: args{c: "E5T6-9UI3-TH15-QR88", logger: logrus.New()},
			want: true,
		},
		{
			name: "fourLower",
			args: args{c: "e5t6-9ui3-th15-qr88", logger: logrus.New()},
			want: true,
		},
		{
			name: "five-too-short",
			args: args{c: "e5t6-9ui3-th15-qr8", logger: logrus.New()},
			want: false,
		},
		{
			name: "six-no-dashes",
			args: args{c: "e5t69ui3th15qr88", logger: logrus.New()},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CodeIsValid(tt.args.c, tt.args.logger); got != tt.want {
				t.Errorf("CodeIsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_nameIsValid(t *testing.T) {
	type args struct {
		n      string
		logger *logrus.Logger
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "valid name - 1",
			args: args{n: "IamAValidName", logger: logrus.New()},
			want: true,
		},
		{
			name: "valid name - 1",
			args: args{n: "I am n valid name", logger: logrus.New()},
			want: true,
		},
		{
			name: "invalid name - 2",
			args: args{n: "I-am-an-invalid-name", logger: logrus.New()},
			want: false,
		},
		{
			name: "valid name - 2",
			args: args{n: "Iam5Valid", logger: logrus.New()},
			want: true,
		},
		{
			name: "valid name with space",
			args: args{n: "I am Valid", logger: logrus.New()},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NameIsValid(tt.args.n, tt.args.logger); got != tt.want {
				t.Errorf("NameIsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDB_Add(t *testing.T) {
	a := assert.New(t)
	db := NewDB(logrus.New())

	// valid item
	newItem := &ProduceItem{
		Name:      "gopher2",
		Code:      "1234-abcd-ABCD-1234",
		UnitPrice: 2.00,
	}

	a.Equal(nil, db.Add(newItem))

	// check name is valid
	newItem2 := &ProduceItem{
		Name:      "!@#$",
		Code:      "1234-abcd-ABCD-1234",
		UnitPrice: 2.00,
	}

	a.Equal(ErrInvalidName, db.Add(newItem2))

	// check code is valid
	newItem3 := &ProduceItem{
		Name:      "pname",
		Code:      "1234",
		UnitPrice: 2.00,
	}

	a.Equal(ErrInvalidCode, db.Add(newItem3))

	// negative unit price - invalid
	newItem5 := &ProduceItem{
		Name:      "myname",
		Code:      "1234-abcd-ABCD-1234",
		UnitPrice: -2.00,
	}

	a.Equal(ErrInvalidUnitPrice, db.Add(newItem5))

	// check price field with too many digits
	newItem4 := &ProduceItem{
		Name:      "pname",
		Code:      "1234-abcd-ABCD-1234",
		UnitPrice: 2.001123,
	}

	db.Add(newItem4)
	pi := GetItemIndex(newItem4.Code, db.Produce, logrus.New())
	if pi == nil {
		a.Fail("pi is nil")
	}
	a.Equal(2.00, db.Produce[*pi].UnitPrice)
}

func TestDB_Get(t *testing.T) {
	a := assert.New(t)
	db := NewDB(logrus.New())

	logger := logrus.New()
	testItems := []*ProduceItem{&ProduceItem{
		Name:      "carrot",
		Code:      "1234-1234-1234-1234",
		UnitPrice: 1.02,
	},
		&ProduceItem{
			Name:      "bean",
			Code:      "2345-2345-2345-2345",
			UnitPrice: 3.55,
		},
		// this item will fail with ErrInvalidCode
		&ProduceItem{
			Name:      "beets",
			Code:      "234@-2345-2345-2345",
			UnitPrice: 3.55,
		},
		&ProduceItem{
			Name:      "corn",
			Code:      "2341-2!45-2345-2345",
			UnitPrice: 3.55,
		},
		&ProduceItem{
			Name:      "corn",
			Code:      "x",
			UnitPrice: 3.55,
		},
	}

	for _, p := range testItems {

		// add item to the db
		db.Add(p)

		if !CodeIsValid(p.Code, logger) {
			_, getErr := db.Get(p.Code)
			a.Equal(ErrInvalidCode, getErr)
		} else {

			i, getErr := db.Get(p.Code)
			a.NoError(getErr)
			a.Equal(p.Code, i.Code)
			a.Equal(p.UnitPrice, i.UnitPrice)
			a.Equal(p.Name, i.Name)

		}

	}

}

func TestDB_List(t *testing.T) {

	a := assert.New(t)

	db := NewDB(logrus.New())

	p := &ProduceItem{
		Name:      "carrot",
		Code:      "1234-1234-1234-1234",
		UnitPrice: 1.02,
	}
	pp := &ProduceItem{
		Name:      "bean",
		Code:      "2345-2345-2345-2345",
		UnitPrice: 3.55,
	}
	db.Add(p)
	db.Add(pp)
	i := db.List()

	a.Equal(p.Name, i[0].Name)
	a.Equal(p.Code, db.Produce[0].Code)
	a.Equal(p.UnitPrice, db.Produce[0].UnitPrice)
	a.Equal(pp.Name, i[1].Name)
	a.Equal(pp.Code, db.Produce[1].Code)
	a.Equal(pp.UnitPrice, db.Produce[1].UnitPrice)

}

func Test_getItemIndex(t *testing.T) {
	type args struct {
		c      string
		pi     []*ProduceItem
		logger *logrus.Logger
	}
	tests := []struct {
		name    string
		args    args
		want    *int
		wantErr bool
	}{
		{
			name: "item found",
			args: args{
				c: "1234-1234-abcd-ABCD",
				pi: []*ProduceItem{
					{
						Name:      "gopher1",
						Code:      "1234-1234-abcd-ABCD",
						UnitPrice: 2.00,
					},
					{
						Name:      "gopher2",
						Code:      "1234-abcd-ABCD-1234",
						UnitPrice: 2.00,
					},
				},
				logger: logrus.New(),
			},
			want:    intP(0),
			wantErr: false,
		},
		{
			name: "item found",
			args: args{
				c: "1234-abcd-ABCD-1234",
				pi: []*ProduceItem{

					{
						Name:      "gopher2",
						Code:      "1234-abcd-ABCD-1234",
						UnitPrice: 2.00,
					},
				},
				logger: logrus.New(),
			},
			want:    intP(0),
			wantErr: false,
		},
		{
			name: "item not found",
			args: args{
				c: "7897-1234-abcd-ABCF",
				pi: []*ProduceItem{
					{
						Name:      "gopher1",
						Code:      "1234-1234-abcd-ABCD",
						UnitPrice: 2.00,
					},
					{
						Name:      "gopher2",
						Code:      "1234-abcd-ABCD-1234",
						UnitPrice: 2.00,
					},
				},
				logger: logrus.New(),
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetItemIndex(tt.args.c, tt.args.pi, tt.args.logger)

			if got == nil && tt.want != nil {
				t.Errorf("GetItemIndex() got = %v, want %v", got, tt.want)

			} else if tt.want != nil && *got != *tt.want {
				t.Errorf("GetItemIndex() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// ExampleNameIsValid - any alphanumeric string without special chars is a valid string
func ExampleNameIsValid() {
	fmt.Println(NameIsValid("IamAValidName000111", logrus.New()))
	// Output: true
}

// ExampleNameIsValid_second - a hyphen is considered invalid.  Only alphanumeric values are valid
func ExampleNameIsValid_second() {
	fmt.Println(NameIsValid("IamAValidName000111-", logrus.New()))
	// Output: false
}

// ExampleCodeIsValid - valid characters are a-zA-Z0-9-.   characters must be in groups of four separated by a hyphen
func ExampleCodeIsValid() {

	fmt.Println(CodeIsValid("aaaa-aaaa-aaaa-aaaa", logrus.New()))
	// Output: true
}

// ExampleCodeIsValid_second - any deviation from acceptable characters or format will result in a false response
func ExampleCodeIsValid_second() {

	fmt.Println(CodeIsValid("$111-2222-3333-4444", logrus.New()))
	// Output: false

}

// ExamplePriceIsValid - prices are zero and positive float64 values up to two decimal places in length.  If a number with
// greater than two digits is passed it, the value will be flagged as valid as it will be truncated to two decimal places.
// Any negative values will be flagged as invalid and return false
func ExamplePriceIsValid() {

	fmt.Println(PriceIsValid(2.22, logrus.New()))
	// Output: true

}

// ExamplePriceIsValid_second - Any negative values will be flagged as invalid and return false
func ExamplePriceIsValid_second() {

	fmt.Println(PriceIsValid(-2.22, logrus.New()))
	// Output: false

}

// intP is a quick helper to return an int  pointer for a given int
func intP(i int) *int {
	return &i
}
