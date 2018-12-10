package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

type ListKeys struct {
	Keys []Key `json:"keys"`
}

type Key struct {
	Key     string `json:"key"`
	Version int    `json:"version"`
}

type ListGroups struct {
	Groups []string `json: "groups"`
}

// GenerateKey generates hyperdrive server key
func GenerateKey(length int) string {
	// Change seed explicitly
	rand.Seed(time.Now().UTC().UnixNano())

	hex := "0123456789ABCDEF"
	ret := ""

	// Build random key
	for i := 0; i < length; i++ {
		index := rand.Intn(len(hex))
		ret = ret + string(hex[index])
	}
	log.Println("Create key: ", ret)
	return ret
}

// PutKey hyperdrive server
func PutKey(hdType string, key, payloadFile string, baseURL string) *http.Request {

	payload, _ := ioutil.ReadFile(payloadFile)
	data := strings.NewReader(string(payload))

	req := &http.Request{}
	var err error

	switch hdType {
	case "server":
		uri := baseURL + "store/" + key
		req, err = http.NewRequest(http.MethodPut, uri, data)

		if err != nil {
			log.Fatal("Put Key, uri=", uri, "error:", err)
		}

		// Payload size is needed
		fi, err2 := os.Stat(payloadFile)

		if err2 != nil {
			log.Fatal("os.Stat() of", payloadFile, "error:", err2)
		}

		// get the size
		size := fi.Size()

		// Set headers value with relevant payload size
		headersValue := fmt.Sprintf("%s%d;", "application/x-scality-storage-data;data=", size)
		req.Header.Set("Content-type", headersValue)

	case "client":
		uri := baseURL + key
		req, err = http.NewRequest(http.MethodPut, uri, data)

		if err != nil {
			log.Fatal("Put Key, uri=", uri, "error:", err)
		}

	default:
		panic("hd-type must be in {server, client}, found: " + hdType)

	}

	return req
}

// GetKeyClient hyperdrive client
func GetKeyClient(key, BaseClient string) *http.Request {
	uri := BaseClient + key
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	log.Println(uri)

	if err != nil {
		log.Fatal(err)
	}

	return req
}

// GetKey hyperdrive server
func GetKey(key, baseserver string) *http.Request {
	uri := baseserver + "store/" + key
	fmt.Println(uri)
	req, err := http.NewRequest(http.MethodGet, uri, nil)

	req.Header.Set("Accept", "application/x-scality-storage-data;meta;usermeta;data")

	if err != nil {
		panic(err)
	}
	return req
}

// DelKey hyperdrive server
func DelKey(key, baseserver string) *http.Request {
	uri := baseserver + key
	fmt.Println(uri)
	req, err := http.NewRequest(http.MethodDelete, uri, nil)

	req.Header.Set("Accept", "application/x-scality-storage-data;meta;usermeta;data")

	if err != nil {
		panic(err)
	}
	return req
}

// DelKeyClient hyperdrive client
func DelKeyClient(key, baseclient string) *http.Request {
	uri := baseclient + key
	req, err := http.NewRequest(http.MethodDelete, uri, nil)
	log.Println(uri)

	if err != nil {
		panic(err)
	}

	return req
}

// Returns an int >= min, < max
func randomInt(min, max int) int {
	return min + rand.Intn(max-min)
}

// RandomString generates a random string of A-Z chars with len = l
func RandomString(len int) string {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		bytes[i] = byte(randomInt(65, 90))
	}
	return string(bytes)
}
