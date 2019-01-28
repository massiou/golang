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

	// Convert payloadFile into string for GET comparisons
	data, err := ioutil.ReadFile(payloadFile)
	if err != nil {
		log.Println(err)
	}
	strData := string(data)

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
		if operation == "get" {
			responseData, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Fatal(err)
			}
			responseString := string(responseData)
			log.Println("Compare PUT and GET payload, expected", payloadFile, "content")
			if responseString != strData {
				log.Fatal("GET response body different from original PUT, expected:", payloadFile)
			}
		} else {
			io.Copy(ioutil.Discard, res.Body)

		}
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
	if len(keysGenerated) != len(keys) {
		fmt.Println("nr keys generated=", len(keysGenerated), "nr keys=", len(keys))
		panic("Keys generated != keys")
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

	// Perform GET operations
	_, throughputGet := performWorkload(hdType, "get", baseserver, keys2, payloadFile, size)

	// Perform DEL operations
	_, throughputDel := performWorkload(hdType, "del", baseserver, keys2, payloadFile, size)

	// Perform GET, expected 404
	client := &http.Client{}
	for _, key := range keys2 {
		log.Println("Reget key:", key)
		opRequest := utils.OpKey(hdType, "get", key, payloadFile, size, baseserver)
		res, err := client.Do(opRequest)
		if err != nil {
			log.Fatal("err=", err)
		}

		if res.StatusCode != 404 {
			log.Fatal("status code=", res.StatusCode, "res=", res)
		}

	}

	thrpt["put"] = throughput
	thrpt["get"] = throughputGet
	thrpt["del"] = throughputDel

	wgMain.Done()
	chanThpt <- thrpt
}

var wgMain sync.WaitGroup

func main() {

	// Arguments
	typePtr := flag.String("hd-type", "server", "Choose between hyperdrive 'server' or 'client'")
	payloadPtr := flag.String("payload-file", "/etc/hosts", "payload file")
	nrinstancesPtr := flag.Int("nrinstances", 1, "number of HD clients/servers")
	nrkeysPtr := flag.Int("nrkeys", 1, "number of keys per goroutine")
	tcKindPtr := flag.String("tc-kind", "", "traffic control kind")
	tcOptionsPtr := flag.String("tc-opt", "", "traffic control options")
	tcPortPtr := flag.Int("tc-port", 0, "traffic control port")
	operations := []string{"put", "get", "del"}
	basePortPtr := flag.Int("port", 4244, "base server port")
	ipaddrPtr := flag.String("ip", "127.0.0.1", "hd base IP address (server or client)")

	flag.Parse()

	// Main call
	chanThrpt := make(chan map[string]float64)

	// Launch goroutines in a loop
	for nrinstances := 0; nrinstances < *nrinstancesPtr; nrinstances++ {
		port := *basePortPtr + nrinstances
		baseURL := "http://" + *ipaddrPtr + ":" + strconv.Itoa(port) + "/"
		wgMain.Add(1)
		go mainFunc(*typePtr, operations, baseURL, *nrkeysPtr, *payloadPtr, &wgMain, chanThrpt)
	}

	// Launch Traffic Control
	if *tcKindPtr != "" && *tcOptionsPtr != "" && *tcPortPtr != 0 {
		utils.TrafficControl(*tcKindPtr, *tcOptionsPtr, *tcPortPtr)
	}
	wgMain.Wait()

	// Get the throughput for each instance
	for nrinstances := 0; nrinstances < *nrinstancesPtr; nrinstances++ {
		//log.Println("Wait for chan")
		thrpt := <-chanThrpt
		log.Println("Instance ID", nrinstances, "Throughput=", thrpt, "Mo/s")
	}

	// Delete Traffic Control
	if *tcKindPtr != "" && *tcOptionsPtr != "" && *tcPortPtr != 0 {
		utils.DeleteTrafficRules()
	}
}
