package utils

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
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

// GenerateKey
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

// PutKey
func PutKey(key, payload string, baseserver string) *http.Request {
	uri := baseserver + "store/" + key
	data := strings.NewReader(payload)
	log.Println(uri)
	req, err := http.NewRequest(http.MethodPut, uri, data)

	headersValue := fmt.Sprintf("%s%d;", "application/x-scality-storage-data;data=", len(payload))
	req.Header.Set("Content-type", headersValue)

	if err != nil {
		panic(err)
	}

	return req
}

// PutKeyClient hyperdrive client
func PutKeyClient(key, payload, baseclient string) *http.Request {
	uri := baseclient + key
	data := strings.NewReader(payload)
	log.Println(uri)
	req, err := http.NewRequest(http.MethodPut, uri, data)

	if err != nil {
		panic(err)
	}
	return req
}

// GetKeyClient hyperdrive client
func GetKeyClient(key, BaseClient string) *http.Request {
	uri := BaseClient + key
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	log.Println(uri)

	if err != nil {
		panic(err)
	}

	return req
}

// GetKey
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

// DelKey
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

// Generate a random string of A-Z chars with len = l
func RandomString(len int) string {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		bytes[i] = byte(randomInt(65, 90))
	}
	return string(bytes)
}
