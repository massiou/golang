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
		hdType      string
		operation   string
		baseurl     string
		key         string
		file        string
		fileSize    int
		expectedRes int
	}

	// Generate use cases for server
	var testworkload []test
	testworkload = append(testworkload, test{"server", "put", "http://127.0.0.1:4244/", "key0", fileTest, size, http.StatusOK})
	testworkload = append(testworkload, test{"server", "get", "http://127.0.0.1:4244/", "key0", fileTest, size, http.StatusOK})
	testworkload = append(testworkload, test{"server", "del", "http://127.0.0.1:4244/", "key0", fileTest, size, http.StatusOK})
	testworkload = append(testworkload, test{"client", "put", "http://127.0.0.1:4244/", "key0", fileTest, size, http.StatusOK})

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
		opRequest := OpKey(cTest.hdType, cTest.operation, cTest.key, cTest.file, cTest.fileSize, cTest.baseurl)
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
