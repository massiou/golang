package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"./utils"
)

const (
	// BaseServer hyperdrive servers url
	BaseServer1 = "http://127.0.0.1:4244/"
	BaseServer2 = "http://127.0.0.1:4245/"
	BaseServer3 = "http://127.0.0.1:4246/"
	// BaseClient Hyperdrive client base url
	BaseClient         = "http://127.0.0.1:18888/"
	PortClient         = 18888
	PortServer         = 4244
	maxFileDescriptors = 1000
)

/*

	log.Println("Client Key=", randomKey)
	log.Println(operation, " key: ", key, "on", baseURL)
	opRequestClient := utils.OpKey(hdType, "get", key, payloadFile, size, baseURL)

	res2, err2 := client.Do(opRequestClient)
	if err2 != nil {
		log.Fatal("err=", err2)
	}

	if res2.StatusCode >= 300 {
		log.Fatal("status code=", res2.StatusCode, "res=", res2)
	}

	io.Copy(ioutil.Discard, res2.Body)

	res2.Body.Close()

*/

// performPut
func performWorkload(
	hdType string,
	operation string,
	baseURL string,
	keys []string,
	payloadFile string) ([]string, float64) {

	client := &http.Client{}

	throughput := 0.0
	size := 0
	var keysGenerated []string
	var totalSize int

	start := time.Now()

	if operation == "put" {
		// Payload size is needed for PUT
		fi, errSize := os.Stat(payloadFile)

		if errSize != nil {
			log.Fatal("os.Stat() of", payloadFile, "error:", errSize)
		}
		// get the size
		size = int(fi.Size())
	}

	// Loop on all keys
	for _, key := range keys {
		// Build request
		log.Println(operation, " key: ", key, "on", baseURL)
		opRequest := utils.OpKey(hdType, operation, key, payloadFile, size, baseURL)
		res, err := client.Do(opRequest)

		if err != nil {
			log.Fatal("err=", err)
		}

		if res.StatusCode >= 300 {
			log.Fatal("status code=", res.StatusCode, "res=", res)
		}

		io.Copy(ioutil.Discard, res.Body)
		res.Body.Close()

		// After a PUT on client hdproxy, get the generated key
		if operation == "put" && hdType == "client" {
			randomKey := res.Header.Get("Scal-Key")
			keysGenerated = append(keysGenerated, randomKey)
		} else {
			keysGenerated = append(keysGenerated, key)
		}

		// Update total put size
		totalSize += int(size)

		// Get elapsed time and convert it from nano to seconds
		elapsed := int(time.Since(start)) / int(math.Pow10(9))

		if elapsed != 0 {
			// in Mo/s
			throughput = float64((totalSize / elapsed) / int(math.Pow10(6)))
			log.Println("operation=", operation, "Throughput: ", throughput, "Mo/s")
			log.Println("totalSize=", totalSize, "nrkeys=", len(keysGenerated), "elapsed=", elapsed)
		}
	}
	return keys, throughput
}

func getKeysIndex(client *http.Client, baseserver string) utils.ListKeys {
	var keys utils.ListKeys

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

func getGroupsIndex(client *http.Client, baseserver string) utils.ListGroups {
	var groups utils.ListGroups

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

//generateKeys
func generateKeys(hdType string, nrkeys int) []string {
	var keys []string

	// Store a random number to identify the current instance
	rand.Seed(time.Now().UTC().UnixNano())
	number := rand.Intn(1000)
	key := "defaultKey"
	for elt := 0; elt < nrkeys; elt++ {
		// Generate key
		if hdType == "server" {
			key = utils.GenerateKey(64)
		} else if hdType == "client" {
			key = fmt.Sprintf("dir-%d/obj-%d", number, elt)
		}
		keys = append(keys, key)
	}
	return keys
}

func mainFunc(hdType string, operations []string, baseserver string, nrkeys int, payloadFile string, wgMain *sync.WaitGroup) {
	defer wgMain.Done()

	// generate nrkeys random keys
	keys := generateKeys(hdType, nrkeys)

	// Perform PUT operations
	keys2, throughput := performWorkload(hdType, "put", baseserver, keys, payloadFile)
	fmt.Println("Operations=", operations, "Throughput=", throughput)

	// Perform GET operations
	_, throughput2 := performWorkload(hdType, "get", baseserver, keys2, payloadFile)
	fmt.Println("Operations=", operations, "Throughput=", throughput2)

	// Perform DEL operations
	_, throughput3 := performWorkload(hdType, "del", baseserver, keys2, payloadFile)
	fmt.Println("Operations=", operations, "Throughput=", throughput3)
}

var wgMain sync.WaitGroup

func main() {

	// Arguments
	typePtr := flag.String("hd-type", "server", "Choose between hyperdrive 'server' or 'client'")
	payloadPtr := flag.String("payload-file", "/etc/hosts", "payload file")
	nrinstancesPtr := flag.Int("nrinstances", 1, "number of HD clients/servers")
	nrkeysPtr := flag.Int("nrkeys", 1, "number of keys per goroutine")

	operations := []string{"put", "get", "del"}

	flag.Parse()

	// Main call
	portBase := 0
	switch *typePtr {
	case "server":
		portBase = PortServer

	case "client":
		portBase = PortClient

	default:
		panic("Please choose hd-type in {server, client}, found: " + *typePtr)
	}

	for nrinstances := 0; nrinstances < *nrinstancesPtr; nrinstances++ {
		port := portBase + nrinstances
		baseURL := "http://127.0.0.1:" + strconv.Itoa(port) + "/"
		wgMain.Add(1)
		go mainFunc(*typePtr, operations, baseURL, *nrkeysPtr, *payloadPtr, &wgMain)

	}
	wgMain.Wait()
}
