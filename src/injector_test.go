package main

import (
	"log"
	"os"
	"testing"
)

func TestPerformPutGet(t *testing.T) {

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

	var testworkload []test
	testworkload = append(testworkload, test{"server", "put", "http://127.0.0.1:4244/", []string{"key0"}, "/etc/hosts", size})
	testworkload = append(testworkload, test{"server", "get", "http://127.0.0.1:4244/", []string{"key0"}, "/etc/hosts", size})
	testworkload = append(testworkload, test{"server", "del", "http://127.0.0.1:4244/", []string{"key0"}, "/etc/hosts", size})

	for _, cTest := range testworkload {

		keys := []string{"key0"}
		keysGenerated, throughput := performWorkload(
			cTest.hdType, cTest.operation, cTest.baseurl, cTest.keys, cTest.file, cTest.fileSize)

		log.Println("keys=", keys, "throughput=", throughput)

		if keysGenerated == nil {
			t.Error("expected:", keys, "found:", keysGenerated)
		}
	}
}
