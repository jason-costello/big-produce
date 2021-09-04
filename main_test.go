package main

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"net/http"
	"runtime"
	"strconv"
	"testing"
)

func Test_setHeader(t *testing.T) {
	a := assert.New(t)
	got := setHeader("one-key", "one-val")
	var want = func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("one-key", "one-key")
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	a.IsType(want, got)
}

//func Test_getServer(t *testing.T) {
//	a := assert.New(t)
//	got := getServer()
//	a.IsType(http.Server{}, got)
//	a.Equal(":8088", got.Addr)
//	a.Equal(5*time.Second, got.IdleTimeout)
//}
//

func Test_loadDB(t *testing.T) {
	a := assert.New(t)

	db, err := LoadDB(logrus.New())
	a.NoError(err)

	if item, err := db.Get("A12T-4GH7-QPL9-3N4M"); err != nil {
		a.Failf("got:  %v    want:  A12T-4GH7-QPL9-3N4M", item.Code)
	}
	if item, err := db.Get("E5T6-9UI3-TH15-QR88"); err != nil {
		a.Failf("got:  %v    want:  E5T6-9UI3-TH15-QR88", item.Code)
	}
	if item, err := db.Get("YRT6-72AS-K736-L4AR"); err != nil {
		a.Failf("got:  %v    want:  YRT6-72AS-K736-L4AR", item.Code)
	}
	if item, err := db.Get("TQ4C-VV6T-75ZX-1RMR"); err != nil {
		a.Failf("got:  %v    want:  TQ4C-VV6T-75ZX-1RMR", item.Code)
	}
	if _, err := db.Get("this-isnt-inDB-fail"); err != nil {
		a.Error(err)
	}
}

func Test_loadLogger(t *testing.T) {

	a := assert.New(t)

	logger := loadLogger("1")
	a.Equal(1, int(logger.Level))
	a.Equal(true, logger.ReportCaller)

	logger = loadLogger("2")
	a.Equal(2, int(logger.Level))
	a.Equal(true, logger.ReportCaller)

	logger = loadLogger("3")
	a.Equal(3, int(logger.Level))
	a.Equal(true, logger.ReportCaller)

	logger = loadLogger("4")
	a.Equal(4, int(logger.Level))
	a.Equal(true, logger.ReportCaller)

	logger = loadLogger("5")
	a.Equal(5, int(logger.Level))
	a.Equal(true, logger.ReportCaller)

	logger = loadLogger("6")
	a.Equal(6, int(logger.Level))
	a.Equal(true, logger.ReportCaller)

	logger = loadLogger("7")
	a.Equal(7, int(logger.Level))
	a.Equal(true, logger.ReportCaller)

}

func Test_loadMaxProcs(t *testing.T) {
	a := assert.New(t)
	got := loadMaxProcs("1")
	a.EqualValues(1, got)

	got = loadMaxProcs("5")
	a.Greater(got, 0)
	a.LessOrEqual(got, 5)
	// a.EqualValues(5, got)

	got = loadMaxProcs("0")
	a.EqualValues(runtime.NumCPU(), got)

	got = loadMaxProcs("-10")
	a.EqualValues(runtime.NumCPU(), got)

	got = loadMaxProcs(strconv.Itoa(runtime.NumCPU() + 1))
	a.EqualValues(runtime.NumCPU(), got)

}

func Test_getSrvAddress(t *testing.T) {

	a := assert.New(t)

	got := getSrvAddress("127.0.0.1", "1234")
	a.Equal("127.0.0.1:1234", got)

	got = getSrvAddress("127.0.0.1", "")
	a.Equal("127.0.0.1:8088", got)

	got = getSrvAddress("255.255.255.255", "1234")
	a.Equal("255.255.255.255:1234", got)

	got = getSrvAddress("256.256.256.256", "1234")
	a.Equal(":1234", got)

	got = getSrvAddress("", "")
	a.Equal(":8088", got)

}
