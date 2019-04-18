package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type listKeys struct {
	Keys []key `json:"keys"`
}

// Key server struct
type key struct {
	Key     string `json:"key"`
	Version int    `json:"version"`
}

type listGroups struct {
	Groups []string `json:"groups"`
}

type hdRequest struct {
	hdType  string // server or client
	request string
	key     string
	file    string
	size    int
	baseURL string
}

func (hdReq hdRequest) String() string {
	return fmt.Sprintf(
		"type:%s, req:%s, key:%s, file:%s, size:%d, url:%s",
		hdReq.hdType, hdReq.request, hdReq.key, hdReq.file, hdReq.size, hdReq.baseURL)
}

// CustomClient Customize the Transport to have larger connection pool
func CustomClient(maxConnections int) *http.Client {
	defaultRoundTripper := http.DefaultTransport
	defaultTransportPointer, ok := defaultRoundTripper.(*http.Transport)
	if !ok {
		panic(fmt.Sprintf("defaultRoundTripper not an *http.Transport"))
	}
	// dereference it to get a copy of the struct that the pointer points to
	defaultTransport := defaultTransportPointer
	defaultTransport.MaxIdleConns = 100
	defaultTransport.MaxIdleConnsPerHost = 100

	client := &http.Client{Transport: defaultTransport}

	return client
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

// OpKey PUT/GET/DELETE function
func OpKey(hdReq hdRequest) *http.Request {

	payload, _ := ioutil.ReadFile(hdReq.file)
	data := strings.NewReader(string(payload))

	req := &http.Request{}
	var err error

	log.Println(hdReq)

	switch hdReq.hdType {
	case "server":
		req, err = opKeyServer(hdReq.request, hdReq.key, hdReq.baseURL, data, hdReq.size)
	case "client":
		req, err = opKeyClient(hdReq.request, hdReq.key, hdReq.baseURL, data, hdReq.size)
	default:
		panic("hd-type must be in {server, client}, found: " + hdReq.hdType)
	}

	if err != nil {
		log.Fatal("error:", err)
	}

	return req
}

func opKeyServer(
	request string,
	key string,
	baseURL string,
	data io.Reader,
	size int) (*http.Request, error) {

	req := &http.Request{}
	var err error
	headersValue := ""
	uri := baseURL + "store/" + key
	switch request {
	case "put":
		req, err = http.NewRequest(http.MethodPut, uri, data)
		// Set headers value with relevant payload size
		headersValue = fmt.Sprintf("%s%d;", "application/x-scality-storage-data;data=", size)
		req.Header.Set("Content-type", headersValue)
	case "get":
		req, err = http.NewRequest(http.MethodGet, uri, nil)
		headersValue = "application/x-scality-storage-data;meta;usermeta;data"
		req.Header.Set("Accept", headersValue)
	case "del":
		req, err = http.NewRequest(http.MethodDelete, uri, nil)
		headersValue = "application/x-scality-storage-data"
		req.Header.Set("Content-type", headersValue)
	default:
		panic("Operation: '" + request + "' not available")
	}
	return req, err
}

func opKeyClient(
	request string,
	key string,
	baseURL string,
	data io.Reader,
	size int) (*http.Request, error) {

	req := &http.Request{}
	var err error
	uri := baseURL + "store/" + key
	switch request {
	case "put":
		req, err = http.NewRequest(http.MethodPost, uri, data)
	case "get":
		req, err = http.NewRequest(http.MethodGet, uri, nil)
	case "del":
		req, err = http.NewRequest(http.MethodDelete, uri, nil)
	default:
		panic("Operation: '" + request + "' not available")
	}
	return req, err
}

// getKeysIndex for hyperdrive server
func getKeysIndex(client *http.Client, baseserver string) listKeys {
	var keys listKeys

	uri := baseserver + "info/index/key/list/"

	req, _ := http.NewRequest(http.MethodGet, uri, nil)

	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)

	json.Unmarshal(body, &keys)

	return keys
}

// getGroupsIndex for hyperdrive server
func getGroupsIndex(client *http.Client, baseserver string) listGroups {
	var groups listGroups

	uri := baseserver + "info/index/group/list/"

	req, _ := http.NewRequest(http.MethodGet, uri, nil)

	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)

	json.Unmarshal(body, &groups)

	return groups
}

// GetKeyClient hyperdrive client
func GetKeyClient(hdType string, key, BaseClient string) *http.Request {
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

//GenerateKeys returns a list of generated keys
func GenerateKeys(hdType string, nrkeys int) []string {
	var keys []string

	// Store a random number to identify the current instance
	rand.Seed(time.Now().UTC().UnixNano())
	number := rand.Intn(1000)
	key := "defaultKey"
	for elt := 0; elt < nrkeys; elt++ {
		// Generate key
		if hdType == "server" {
			key = GenerateKey(64)
		} else if hdType == "client" {
			key = fmt.Sprintf("dir-%d/obj-%d", number, elt)
		}
		keys = append(keys, key)
	}
	return keys
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

// TrafficControl uses tc and netem to simulate network issue on a specific port
func TrafficControl(qdiscKind, options string, port int) bool {
	out1, err1 := exec.Command("/bin/sh", "-c", "/sbin/tc qdisc add dev lo root handle 1: prio priomap 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0").Output()

	if err1 != nil {
		DeleteTrafficRules()
		log.Println("tc qdisc1", out1)
		log.Fatal(err1)
	}

	cmdNetem := fmt.Sprintf("tc qdisc add dev lo parent 1:2 handle 20: netem %s %s", qdiscKind, options)
	log.Println(cmdNetem)
	_, err2 := exec.Command("/bin/sh", "-c", cmdNetem).Output()

	if err2 != nil {
		DeleteTrafficRules()
		log.Fatal(err2)
	}

	cmdFilter := fmt.Sprintf("tc filter add dev lo parent 1:0 protocol ip u32 match ip sport %d 0xffff flowid 1:2", port)
	log.Println(cmdFilter)
	_, err3 := exec.Command("/bin/sh", "-c", cmdFilter).Output()

	if err3 != nil {
		DeleteTrafficRules()
		log.Fatal(err3)
	}

	return true
}

// DeleteTrafficRules deletes all tc rules on lo interface
func DeleteTrafficRules() bool {
	_, err := exec.Command("/bin/sh", "-c", "tc qdisc del dev lo root").Output()

	if err != nil {
		log.Fatal(err)
	}

	return true

}
