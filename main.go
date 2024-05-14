package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/sirupsen/logrus"
)

func main() {

	srv := getServer()
	// start the server
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}

}

// getServer grabs the expected env variables, generates the logger instance,
// the database, the router, and finally returns the server
func getServer() http.Server {
	// grab env vars for max procs and set maxProcs
	maxProcsStr := os.Getenv("MAXPROCS")
	maxProcs := loadMaxProcs(maxProcsStr)

	// grab env var for log level and get a logger
	logLevelStr := os.Getenv("LOGLEVEL")
	logger := loadLogger(logLevelStr)

	addr := getSrvAddress(os.Getenv("ADDRESS"), os.Getenv("PORT"))

	// setup db, handlers, and routes
	var db *DB
	var err error
	if db, err = LoadDB(logger); err != nil {
		panic(err)
	}
	h := NewHandler(db, maxProcs, logger)
	r := LoadRouter(h)

	return http.Server{
		Addr:        addr,
		IdleTimeout: 5 * time.Second,
		ErrorLog:    log.Default(),
		Handler:     r,
	}
}

// LoadDB grabs a new database and fills it with the required produce items
func LoadDB(l *logrus.Logger) (*DB, error) {

	db := NewDB(l)

	if err := db.Add(&ProduceItem{Code: "A12T-4GH7-QPL9-3N4M", Name: "Lettuce", UnitPrice: 3.46}); err != nil {
		return nil, err
	}
	if err := db.Add(&ProduceItem{Code: "E5T6-9UI3-TH15-QR88", Name: "Peach", UnitPrice: 2.99}); err != nil {
		return nil, err
	}
	if err := db.Add(&ProduceItem{Code: "YRT6-72AS-K736-L4AR", Name: "Green Pepper", UnitPrice: 0.79}); err != nil {
		return nil, err
	}
	if err := db.Add(&ProduceItem{Code: "TQ4C-VV6T-75ZX-1RMR", Name: "Gala Apple", UnitPrice: 3.59}); err != nil {
		return nil, err
	}

	return db, nil
}

// loadLogger - logging level can be set by env var that is an int
// with a value between 1 and 7.  If not set, value is defaulted to
// 3, error level logging.
//
//	1=PanicLevel,2=FatalLevel,3=ErrorLevel,4=WarnLevel,5=InfoLevel,6=DebugLevel,7=TraceLevel
//
// returns pointer to logger
func loadLogger(logLevelStr string) *logrus.Logger {
	logLevel, err := strconv.ParseUint(logLevelStr, 10, 32)
	if err != nil {
		logLevel = 3
	}
	logger := logrus.New()
	logger.SetLevel(logrus.Level(logLevel))
	logger.ReportCaller = true
	return logger
}

// loadMaxProcs sets the number of procs to use for concurrency operations
// if the env var is not specified it will be set to the runtime cpu count.
// valid values must be > 0 and <= runtime.NumCPU, if not, it will be defaulted
// to runtime.NumCPU
// k8s and openshift labels can be used on a pod to set this value as well.
func loadMaxProcs(maxProcsStr string) int {
	maxProcs, err := strconv.Atoi(maxProcsStr)
	if err != nil || maxProcs <= 0 || maxProcs > runtime.NumCPU() {
		maxProcs = runtime.NumCPU()
	}

	return maxProcs

}

// getSrvAddress just takes in a string representing the desired address and port to run the
// webserver on.
// If address is empty or a malformed IP (IPV4) the default is blank ("")
// If port is empty or non-numeric, default to port 8088
func getSrvAddress(address string, port string) string {

	var err error
	var pi int

	if pi, err = strconv.Atoi(port); err != nil {
		pi = 8088
	}
	port = strconv.Itoa(pi)

	ipRegex := regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)
	if !ipRegex.MatchString(address) {
		address = ""
	}
	return fmt.Sprintf("%s:%s", address, port)

}

// LoadRouter takes in a Handler pointer, sets up the middleware, and builds the routes.
func LoadRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()

	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
	r.Use(cors.Handler)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Heartbeat("/ping"))
	setHeader("X-XSS-Protection", "1; mode=block")
	setHeader("X-Frame-Options", "deny")

	r.Route("/api/v1/produce", func(r chi.Router) {
		r.With(produceCodeMW).Route("/{code}", func(r chi.Router) {
			r.Get("/", h.GetProduce)
			r.Delete("/", h.DeleteProduce)
		})
		r.Get("/", h.GetAllProduce)
		r.Post("/", h.AddProduce)
	})

	return r
}

func produceCodeMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		produceCode := chi.URLParam(r, "code")

		if produceCode == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("empty produce code"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// setHeader is a helper handler to set a response header key/value
func setHeader(key, value string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(key, value)
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
