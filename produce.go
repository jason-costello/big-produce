package main

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// ErrNotFound will be used when specified items are not found
var ErrNotFound = errors.New("item not found")

// ErrDuplicateItem will be used when an item is trying to be added where the product code of that
// item  already exists in the database
var ErrDuplicateItem = errors.New("item already exists")

// ErrInvalidCode indicates the produce code doesn't meet the required formatting or character
// constraints (0-9a-zA-Z-)  Required format is xxxx-xxxx-xxxx-xxxx
var ErrInvalidCode = errors.New("item code is invalid")

// ErrInvalidName indicates the produce name doesn't meet the required formatting or character
// constraints of being alphanumeric with no other characters allowed (0-9a-zA-Z)
var ErrInvalidName = errors.New("item name is invalid")

// ErrInvalidUnitPrice indicates the produce unit price is less than 0 (negative value).
var ErrInvalidUnitPrice = errors.New("item unit price is invalid")

// ProduceItem represents a single piece of Produce sold by the store.
// The Produce includes name, Produce code, and unit price
type ProduceItem struct {
	// Name is alphanumeric and case-insensitive
	Name string `json:"produce_name"`
	// Code is a sixteen character (plus four dashes) long string with dashes separating each four character group.
	// The codes are alphanumeric and case-insensitive.
	Code string `json:"produce_code"`
	// UnitPrice is a number with up to two decimal places
	UnitPrice float64 `json:"produce_unit_price"`
}

// DB is an in-memory store to track Produce for the store.
type DB struct {
	// Produce is the slice that contains the Produce items being manipulated.
	// This acts as the storage of the database
	Produce []*ProduceItem
	// logger is a local logger instance for the db
	logger *logrus.Logger
	// mtx is a mutex used to lock and unlock Produce to ensure concurrent safety.
	mtx *sync.Mutex
}

// NewDB returns a new, clean db
func NewDB(logger *logrus.Logger) *DB {

	return &DB{
		Produce: []*ProduceItem{},
		logger:  logger,
		mtx:     &sync.Mutex{},
	}
}

// List returns all Produce items in the database.
// If there are no items in the database an empty slice is returned with a nil error
func (d *DB) List() []*ProduceItem {
	return d.Produce

}

// Get returns the item with the passed code
// If the item is not found a ErrNotFound is returned
func (d *DB) Get(code string) (*ProduceItem, error) {

	if !CodeIsValid(code, d.logger) {
		return nil, ErrInvalidCode
	}

	var idx *int
	if idx = GetItemIndex(code, d.Produce, d.logger); idx == nil {
		return &ProduceItem{}, ErrNotFound
	}
	d.mtx.Lock()
	defer d.mtx.Unlock()
	return d.Produce[*idx], nil

}

// Delete will look  for matching code in db and
// if found remove the item.  If the produce code is not
// found in the database an ErrNotFound error is returned.
func (d *DB) Delete(code string) error {

	idx := GetItemIndex(code, d.Produce, d.logger)
	if idx == nil {
		return ErrNotFound
	}

	d.mtx.Lock()
	d.Produce[*idx] = d.Produce[len(d.Produce)-1]
	d.Produce[len(d.Produce)-1] = nil
	d.Produce = d.Produce[:len(d.Produce)-1]
	d.mtx.Unlock()

	return nil

}

// Add creates new items in the database
// and returns ErrDuplicateItem if an item
// already exists with the same code.
func (d *DB) Add(p *ProduceItem) error {

	// check for valid code
	if !CodeIsValid(p.Code, d.logger) {
		return ErrInvalidCode
	}

	// check if name is valid
	if !NameIsValid(p.Name, d.logger) {
		return ErrInvalidName
	}

	// check if price is valid
	var err error
	if !PriceIsValid(p.UnitPrice, d.logger) {
		return ErrInvalidUnitPrice
	}

	// ensure the produce unit price is at max two decimal places.
	p.UnitPrice, err = strconv.ParseFloat(fmt.Sprintf("%0.2f", p.UnitPrice), 64)
	if err != nil {
		return ErrInvalidUnitPrice
	}

	// do a quick check to see if the produce code is already in the database
	if GetItemIndex(p.Code, d.Produce, d.logger) != nil {
		return ErrDuplicateItem
	}

	// lock, write, unlock
	d.mtx.Lock()
	d.Produce = append(d.Produce, p)
	d.mtx.Unlock()

	return nil
}

// PriceIsValid ensures the unit price is a non-negative
// float64
func PriceIsValid(p float64, logger *logrus.Logger) bool {
	_, err := strconv.ParseFloat(fmt.Sprintf("%0.2f", p), 64)
	if p < 0 || err != nil {
		logger.Debugf("p<0::%v || err != nil:: %s", p, err)
		return false
	}
	return true

}

// NameIsValid verifies that the name for a produce item is
// alphanumeric and case-insensitive.
func NameIsValid(n string, logger *logrus.Logger) bool {
	match, err := regexp.MatchString(`^[a-zA-Z0-9\s]+$`, n)
	if err != nil {
		logger.Debug("ERR: name validation error: ", n, match, err.Error())
		return false
	}
	return match
}

// CodeIsValid verifies that all produce codes are case-insensitive, alphanumeric strings
// that are in four groups of four with each group separated by a hyphen.
// valid code example: aaa1-bbb2-ccc3-ddd4
func CodeIsValid(c string, logger *logrus.Logger) bool {
	match, err := regexp.MatchString("[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{4}", c)
	if err != nil {
		logger.Debug("ERR: code validation error: ", c, match, err.Error())
		return false
	}
	return match
}

// GetItemIndex returns a pointer to the zero-based index for the position the produce item is found
// if no produce item is found, a nil value is returned.
func GetItemIndex(c string, pi []*ProduceItem, logger *logrus.Logger) *int {
	for i := 0; i < len(pi); i++ {
		if strings.ToLower(c) == strings.ToLower(pi[i].Code) {
			logger.Debugf("code %s found at index %d", c, i)
			return &i
		}
	}

	return nil
}
