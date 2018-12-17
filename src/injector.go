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

// performWorkload
func performWorkload(
	hdType string,
	operation string,
	baseURL string,
	keys []string,
	payloadFile string,
	size int) ([]string, float64) {

	client := &http.Client{}

	throughput := 0.0
	var keysGenerated []string
	var totalSize int

	start := time.Now()

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

		// Consume the response & Close the request
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
	return keysGenerated, throughput
}

// getKeysIndex for hyperdrive server
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

// getGroupsIndex for hyperdrive server
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

func mainFunc(
	hdType string,
	operations []string,
	baseserver string,
	nrkeys int,
	payloadFile string,
	wgMain *sync.WaitGroup,
	chanThpt chan map[string]float64) {

	defer wgMain.Done()
	thrpt := make(map[string]float64)

	// Payload size is needed for PUT header and to compute throughput
	fi, errSize := os.Stat(payloadFile)

	if errSize != nil {
		log.Fatal("os.Stat() of", payloadFile, "error:", errSize)
	}
	// get the size
	size := int(fi.Size())

	// generate nrkeys random keys
	keys := generateKeys(hdType, nrkeys)

	// Perform PUT operations
	keys2, throughput := performWorkload(hdType, "put", baseserver, keys, payloadFile, size)
	log.Println("Operations=", operations, "Throughput=", throughput)

	// Perform GET operations
	_, throughputGet := performWorkload(hdType, "get", baseserver, keys2, payloadFile, size)
	log.Println("Operations=", operations, "Throughput=", throughputGet)

	// Perform DEL operations
	_, throughputDel := performWorkload(hdType, "del", baseserver, keys2, payloadFile, size)
	log.Println("Operations=", operations, "Throughput=", throughputDel)

	thrpt["put"] = throughput
	thrpt["get"] = throughputGet
	thrpt["del"] = throughputDel

	chanThpt <- thrpt
}

var wgMain sync.WaitGroup

func main() {

	// Arguments
	typePtr := flag.String("hd-type", "server", "Choose between hyperdrive 'server' or 'client'")
	payloadPtr := flag.String("payload-file", "/etc/hosts", "payload file")
	nrinstancesPtr := flag.Int("nrinstances", 1, "number of HD clients/servers")
	nrkeysPtr := flag.Int("nrkeys", 1, "number of keys per goroutine")

	operations := []string{"put", "get", "del"}

	// Throughput computation channel
	chanThrpt := make(chan map[string]float64)

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
		go mainFunc(*typePtr, operations, baseURL, *nrkeysPtr, *payloadPtr, &wgMain, chanThrpt)
		thrpt := <-chanThrpt
		log.Println("Instance ID", nrinstances, "Throughput=", thrpt, "Mo/s")
	}
	wgMain.Wait()
}
