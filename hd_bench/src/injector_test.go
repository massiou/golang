package main

import (
	"log"
	"os"
	"testing"
)

func TestOpKey(t *testing.T) {

	// Test file path and size
	fileTest := "/etc/hosts"
	fi, _ := os.Stat(fileTest)
	size := int(fi.Size())

	type test struct {
		hdType    string
		operation string
		baseurl   string
		key       string
		file      string
		fileSize  int
	}

	// Generate use cases for server
	var testworkload []test
	testworkload = append(testworkload, test{"server", "put", "http://127.0.0.1:4244/", "key0", fileTest, size})
	testworkload = append(testworkload, test{"server", "get", "http://127.0.0.1:4244/", "key0", fileTest, size})
	testworkload = append(testworkload, test{"server", "del", "http://127.0.0.1:4244/", "key0", fileTest, size})

	client := CustomClient(100) // HTTP client
	var errors int

	// Execute use cases
	for _, cTest := range testworkload {
		opRequest := OpKey(cTest.hdType, cTest.operation, cTest.key, cTest.file, cTest.fileSize, cTest.baseurl)
		res, err := client.Do(opRequest)

		if err != nil {
			log.Fatal("err=", err)
		}
		if res.StatusCode >= 300 {
			log.Println("status code=", res.StatusCode, "res=", res)
			errors++
		}

	}
}
