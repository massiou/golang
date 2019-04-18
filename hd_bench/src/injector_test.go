package main

import (
	"log"
	"net/http"
	"os"
	"testing"
)

type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return &http.Response{}, nil
}

func TestOpKey(t *testing.T) {

	// Test file path and size
	fileTest := "/etc/hosts"
	fi, _ := os.Stat(fileTest)
	size := int(fi.Size())

	type test struct {
		hdReq       hdRequest
		expectedRes int
	}

	// Generate use cases for server
	var testworkload []test

	hdServerPut := hdRequest{"server", "put", "key0", fileTest, size, "http://127.0.0.1:4244/"}
	hdServerGet := hdRequest{"server", "get", "key0", fileTest, size, "http://127.0.0.1:4244/"}
	hdServerDel := hdRequest{"server", "del", "key0", fileTest, size, "http://127.0.0.1:4244/"}
	hdClientPut := hdRequest{"client", "put", "key0", fileTest, size, "http://127.0.0.1:4244/"}

	testworkload = append(testworkload, test{hdServerPut, http.StatusOK})
	testworkload = append(testworkload, test{hdServerGet, http.StatusOK})
	testworkload = append(testworkload, test{hdServerDel, http.StatusOK})
	testworkload = append(testworkload, test{hdClientPut, http.StatusOK})

	client := &MockClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
			}, nil
		},
	}
	var errors int

	// Execute use cases
	for _, cTest := range testworkload {
		opRequest := OpKey(cTest.hdReq)
		res, err := client.Do(opRequest)

		if err != nil {
			log.Fatal("err=", err)
		}
		if res.StatusCode != cTest.expectedRes {
			log.Fatal("status code=", res.StatusCode, "res=", res)
			errors++
		}

	}
}
