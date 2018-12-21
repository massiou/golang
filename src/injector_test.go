package main

import (
	"log"
	"os"
	"testing"
)

func TestPerformWorkload(t *testing.T) {

	// Test file path and size
	fileTest := "/etc/hosts"
	fi, _ := os.Stat(fileTest)
	size := int(fi.Size())

	type test struct {
		hdType    string
		operation string
		baseurl   string
		keys      []string
		file      string
		fileSize  int
	}

	// Generate use cases for server
	var testworkload []test
	testworkload = append(testworkload, test{"server", "put", "http://127.0.0.1:4244/", []string{"key0"}, fileTest, size})
	testworkload = append(testworkload, test{"server", "get", "http://127.0.0.1:4244/", []string{"key0"}, fileTest, size})
	testworkload = append(testworkload, test{"server", "del", "http://127.0.0.1:4244/", []string{"key0"}, fileTest, size})

	// Execute use cases
	for _, cTest := range testworkload {
		keysGenerated, throughput := performWorkload(
			cTest.hdType, cTest.operation, cTest.baseurl, cTest.keys, cTest.file, cTest.fileSize)

		log.Println("keys=", cTest.keys, "throughput=", throughput)

		if keysGenerated == nil {
			t.Error("expected:", cTest.keys, "found:", keysGenerated)
		}
	}
}
