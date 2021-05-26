package s3gof3r

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func NewFakeGetter(testurl string) (io.ReadCloser, error) {
	s3obj := New("s3.amazonaws.com", Keys{})
	b := s3obj.Bucket("foobucket")
	u, err := url.Parse(testurl)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse url: %s", testurl)
	}
	c := b.conf()
	c.NTry = 1
	g, _, err := newGetter(*u, c, b)
	if err != nil {
		return nil, fmt.Errorf("newGetter() %s", err)
	}
	return g, nil
}

// Verify graceful recovery (don't hang) when we receive the target
// content length but subsequent chunk requests fail.
func TestFailedGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "2") // trigger unexpected EOF in the client
		fmt.Fprintln(w, "")
	}))
	defer ts.Close()

	g, err := NewFakeGetter(ts.URL)
	if err != nil {
		t.Error(err)
	}
	defer g.Close()

	_, err = ioutil.ReadAll(g)
	if err == nil {
		t.Error("Expected ReadAll() to return an error")
	}
}

// Verify successful read when everything is working correctly.
func TestGetterHappyPath(t *testing.T) {
	expStr := "happy test"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, expStr)
	}))
	defer ts.Close()

	g, err := NewFakeGetter(ts.URL)
	if err != nil {
		t.Error(err)
	}
	defer g.Close()

	d, err := ioutil.ReadAll(g)
	if err != nil {
		t.Error("ReadAll():", err)
	}
	if string(d[:len(d)-1]) != expStr { // strip trailing newline
		t.Errorf("Expected data to be: '%v'. Actual: '%v'", expStr, d)
	}
}
