// Package utils supplies logging and testing utils.
package utils

import (
	"errors"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	iter8utils "github.com/iter8-tools/etc3/util"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

var logLevel logrus.Level = logrus.InfoLevel

// SetLogLevel sets level for logging.
func SetLogLevel(l logrus.Level) {
	logLevel = l
	if log != nil {
		log.SetLevel(logLevel)
	}
}

// GetLogger returns a logger, if needed after creating it.
func GetLogger() *logrus.Logger {
	if log == nil {
		log = logrus.New()
		log.SetLevel(logLevel)
	}
	return log
}

// CompletePath determines complete path of a file
var CompletePath func(prefix string, suffix string) string = iter8utils.CompletePath

// Int32Pointer takes an int32 as input, creates a new variable with the input value, and returns a pointer to the variable
func Int32Pointer(i int32) *int32 {
	return &i
}

// Float32Pointer takes an float32 as input, creates a new variable with the input value, and returns a pointer to the variable
func Float32Pointer(f float32) *float32 {
	return &f
}

// StringPointer takes a string as input, creates a new variable with the input value, and returns a pointer to the variable
func StringPointer(s string) *string {
	return &s
}

type HTTPMethod string

const (
	GET  HTTPMethod = "GET"
	POST            = "POST"
)

// HTTPMethodPointer takes an HTTPMethod as input, creates a new variable with the input value, and returns a pointer to the variable
func HTTPMethodPointer(h HTTPMethod) *HTTPMethod {
	return &h
}

// WaitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
// See https://stackoverflow.com/questions/32840687/timeout-for-waitgroup-wait
func WaitTimeoutOrError(wg *sync.WaitGroup, timeout time.Duration, errCh chan error) error {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return nil // completed normally
	case <-time.After(timeout):
		return errors.New("Timedout waiting for fortio data collection") // timed out
	case err := <-errCh:
		return err
	}
}

// GetJSONBytes downloads JSON from URL and returns a byte slice
func GetJsonBytes(url string) ([]byte, error) {
	var myClient = &http.Client{Timeout: 10 * time.Second}
	r, err := myClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	return body, err
}
