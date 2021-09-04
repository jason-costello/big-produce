package main

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"net/http"
	"sync"
)

// Handler provides access to all handler funcs
type Handler struct {
	DB       *DB
	maxProcs int
	logger   *logrus.Logger
}

// NewHandler returns a pointer to a handler
func NewHandler(db *DB, maxProcs int, logger *logrus.Logger) *Handler {
	return &Handler{DB: db, maxProcs: maxProcs, logger: logger}
}

// handlerErrorLogger - a helper to format debugging log output for handler functions
func handlerErrorLogger(r *http.Request, err error, logger *logrus.Logger) {
	logger.Errorf("host:%sr, url:%s. headers:%v, method:%s, reqAddr:%s, err: %s", r.Host, r.URL, r.Header, r.Method, r.RemoteAddr, err.Error())

}

// GetAllProduce will return a json string with all items from the database
func (h *Handler) GetAllProduce(w http.ResponseWriter, r *http.Request) {

	p := h.DB.List()

	dat, err := json.Marshal(p)
	if err != nil {
		handlerErrorLogger(r, err, h.logger)
		http.Error(w, "error listing produce", http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")

	w.WriteHeader(200)
	w.Write(dat)

}

// GetProduce requires a path variable for the produce code.   The code is searched
// against the database.  If the item is found a json representation of that object is returned.
// If no item is found with  that id a 404 error is returned with "item not found" text.
// If the code is empty a 400 bad request is returned with "verify Produce code" text.
func (h *Handler) GetProduce(w http.ResponseWriter, r *http.Request) {

	code := chi.URLParam(r, "code")

	p, err := h.DB.Get(code)
	if err != nil {
		handlerErrorLogger(r, err, h.logger)

		if err == ErrNotFound {

			http.Error(w, "item not found", 404)
			return
		}

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dat, err := json.Marshal(p)
	if err != nil {
		handlerErrorLogger(r, err, h.logger)
		http.Error(w, "error generating json data", http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}

// DeleteProduce removes a produce item from the database where the code matches the item in the db.
// A path variable for the produce code is required.  If the item is not found a 404 is returned.  if the code
// provided isn't valid a 400 bad request is returned.   If the item is deleted a 204 is returned.
func (h *Handler) DeleteProduce(w http.ResponseWriter, r *http.Request) {

	code := chi.URLParam(r, "code")

	if err := h.DB.Delete(code); err != nil {

		handlerErrorLogger(r, err, h.logger)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(204)

}

// AddResult is used to track the status of a ProduceItem to the database
type AddResult struct {
	// Produce is the ProduceItem to be added
	Produce ProduceItem `json:"produce"`
	// StatusCode is a http status code indicating the status of adding the ProduceItem
	StatusCode int `json:"status_code"`
	// Status is a string representation of the status of adding the ProduceItem.  It contains
	// a string value of the status code and a string description
	Status string `json:"status"`
}

// AddResults is used to track the status of adding multiple ProduceItems to the database in one handler call
type AddResults struct {
	// Results contains all results for adding multiple Produce Items to the db.
	Results []AddResult `json:"results"`
}

// AddProduce adds ProduceItems to the database.   It accepts an array of ProduceItems in json format.
// Prior to adding each item to the database the ProduceItems are checked to validate their name, code, and
// unit prices.  The database is also checked to ensure an existing record does not exist for an item
// with the same code.  The items are added to the database concurrently utilizing the maxProcs variable
// as the number of concurrent items to process.   The pipeline was created in a way to easily insert or remove
// new functions if needed.   The fan-in function is somewhat boilerplate code used to consolidate all channels
// back down to one slice.
func (h *Handler) AddProduce(w http.ResponseWriter, r *http.Request) {

	var pi []ProduceItem

	err := json.NewDecoder(r.Body).Decode(&pi)
	if err != nil {
		handlerErrorLogger(r, err, h.logger)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// generator - takes the produceItems from the Post request and puts them on a channel.
	gen := func(done <-chan interface{}, produceItems ...ProduceItem) <-chan AddResult {
		resultStream := make(chan AddResult)
		go func() {
			defer close(resultStream)

			for _, pi := range produceItems {
				result := AddResult{
					Produce:    pi,
					StatusCode: 0,
					Status:     "",
				}
				select {
				case <-done:
					return
				case resultStream <- result:
				}
			}
		}()
		return resultStream
	} // end gen func

	// verify produce name
	verifyName := func(done <-chan interface{}, incomingStream <-chan AddResult) <-chan AddResult {
		verifiedNameStream := make(chan AddResult)

		go func() {
			defer close(verifiedNameStream)

			for i := range incomingStream {

				if i.StatusCode == 0 {
					if !NameIsValid(i.Produce.Name, h.logger) {
						i.StatusCode = 400
						i.Status = "400: ErrInvalidName"
					}
				}

				select {

				case <-done:
					return

				case verifiedNameStream <- i:
				}
			}
		}()
		return verifiedNameStream
	} // end verifyName func

	// verify produce code
	verifyCode := func(done <-chan interface{}, incomingStream <-chan AddResult) <-chan AddResult {
		verifiedCodeStream := make(chan AddResult)

		go func() {
			defer close(verifiedCodeStream)

			for i := range incomingStream {

				if i.StatusCode == 0 {
					if !CodeIsValid(i.Produce.Code, h.logger) {
						h.logger.Error("code is not valid: ", CodeIsValid(i.Produce.Code, h.logger))
						i.StatusCode = 400
						i.Status = "400: ErrInvalidCode"
					}
				}

				select {
				case <-done:
					return
				case verifiedCodeStream <- i:

				}
			}
		}()
		return verifiedCodeStream
	} // end verifyCode func

	// verify produce unit price
	verifyPrice := func(done <-chan interface{}, incomingStream <-chan AddResult) <-chan AddResult {
		verifiedPriceStream := make(chan AddResult)

		go func() {
			defer close(verifiedPriceStream)

			for i := range incomingStream {

				if i.StatusCode == 0 {
					if !PriceIsValid(i.Produce.UnitPrice, h.logger) {
						i.StatusCode = 400
						i.Status = "400: ErrInvalidPrice"
					}
				}

				select {

				case <-done:
					return

				case verifiedPriceStream <- i:
				}
			}
		}()
		return verifiedPriceStream
	} // end verifyPrice func

	// pipeline stage that attempts to add the incoming ProduceItem to the database.
	// if any errors are returned from the Add function,
	add := func(done <-chan interface{}, incomingStream <-chan AddResult) <-chan AddResult {
		completedWorkStream := make(chan AddResult)

		go func() {
			defer close(completedWorkStream)

			for i := range incomingStream {

				if i.StatusCode == 0 {
					if err := h.DB.Add(&i.Produce); err != nil {
						if errors.Is(err, ErrInvalidCode) {
							i.StatusCode = 400
							i.Status = "400 " + err.Error()
						} else if errors.Is(err, ErrDuplicateItem) {

							i.StatusCode = 409
							i.Status = "409: " + ErrDuplicateItem.Error()
						} else if errors.Is(err, ErrInvalidName) {

							i.StatusCode = 400
							i.Status = "400: " + ErrInvalidName.Error()

						} else {
							i.StatusCode = 500
							i.Status = "500: " + err.Error()
						}

					} else {
						i.StatusCode = 201
						i.Status = "201: added"
					}
				}
				select {

				case <-done:
					return

				case completedWorkStream <- i:
				}
			}
		}()
		return completedWorkStream
	} // end add func

	// fan-in implementation to consolidate the result channels
	fanIn := func(done <-chan interface{}, channels ...<-chan AddResult) <-chan AddResult {
		var wg sync.WaitGroup
		mplexStream := make(chan AddResult)

		mplex := func(c <-chan AddResult) {
			defer wg.Done()
			for i := range c {
				select {
				case <-done:
					return
				case mplexStream <- i:
				}
			}
		}
		wg.Add(len(channels))
		for _, c := range channels {
			go mplex(c)
		}

		go func() {
			wg.Wait()
			close(mplexStream)
		}()

		return mplexStream
	} // end fan-in

	// setup for pipeline
	done := make(chan interface{})
	defer close(done)
	resultsStream := gen(done, pi...)

	// create a slice of channels equal in length to the number of maxProcs
	produceProcessors := make([]<-chan AddResult, h.maxProcs)

	// iterate over maxProcs creating pipelines at each iteration
	for i := 0; i < h.maxProcs; i++ {
		produceProcessors[i] = verifyName(done, verifyCode(done, verifyPrice(done, add(done, resultsStream))))
	}

	// rs will hold the final consolidated results from all channels
	rs := AddResults{}

	// fanIn used to consolidate all the results to rs
	for x := range fanIn(done, produceProcessors...) {
		rs.Results = append(rs.Results, x)
	}

	// marshal results and return
	d, err := json.Marshal(rs)
	if err != nil {
		handlerErrorLogger(r, err, h.logger)
		http.Error(w, "error generating json data", http.StatusBadRequest)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(200)
	w.Write(d)

}
