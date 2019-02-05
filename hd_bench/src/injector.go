package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"

	"./utils"
)

// performWorkload
func performWorkload(
	hdType string, // server or client
	operation string, // PUT / GET / DELETE
	baseURL string,
	keys []string, // list of keys
	payloadFile string,
	size int) ([]string, float64) {

	client := utils.CustomClient(100) // HTTP client

	throughput := 0.0
	var keysGenerated []string
	var totalSize int

	start := time.Now()

	// Loop on all keys
	for _, key := range keys {
		// Build request
		glog.V(2).Info(operation, " key: ", key, " on ", baseURL)
		opRequest := utils.OpKey(hdType, operation, key, payloadFile, size, baseURL)
		res, err := client.Do(opRequest)

		if err != nil {
			log.Fatal("err=", err)
		}
		if res.StatusCode >= 300 {
			log.Fatal("status code=", res.StatusCode, "res=", res)
		}
		// Compare PUT and GET answer
		if operation == "get" {
			comparison := compareGetPut(payloadFile, res)
			if comparison == false {
				log.Fatal("GET response body different from original PUT, expected:", payloadFile)
			}
		} else {
			io.Copy(ioutil.Discard, res.Body)
		}

		// Close the current request
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
		elapsed := float64(time.Since(start)) / (math.Pow10(9))

		if elapsed != 0 {
			// in Mo/s
			throughput = float64(totalSize) / elapsed / float64(math.Pow10(6))
			glog.V(2).Info("operation=", operation, " Throughput: ", throughput, " Mo/s")
			glog.V(2).Info("totalSize=", totalSize, " nrkeys=", len(keysGenerated), " elapsed=", elapsed)
		}
	}
	if len(keysGenerated) != len(keys) {
		fmt.Println("nr keys generated=", len(keysGenerated), "nr keys=", len(keys))
		panic("Keys generated != keys")
	}
	return keysGenerated, throughput
}

func compareGetPut(file string, res *http.Response) bool {
	// Convert payloadFile into string for GET comparisons
	data, err := ioutil.ReadFile(file)
	if err != nil {
		log.Println(err)
	}
	strData := string(data)
	// Consume the response
	responseData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	// Return comparison status
	return string(responseData) == strData
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
	keys := utils.GenerateKeys(hdType, nrkeys)

	var throughput float64
	var keysPut []string

	// put operation is mandatory
	keysPut, throughput = performWorkload(hdType, "put", baseserver, keys, payloadFile, size)
	thrpt["put"] = throughput

	// Loop on optional operations 'GET' and/or 'DELETE'
	for i := 0; i < len(operations); i++ {
		// Perform operations
		operation := operations[i]
		if operation == "get" || operation == "del" {
			_, throughput = performWorkload(hdType, operations[i], baseserver, keysPut, payloadFile, size)
			thrpt[operations[i]] = throughput
		}
	}
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
	operationsPtr := flag.String("operations", "", "worload operations 'get' or 'del' or 'get del', etc")
	basePortPtr := flag.Int("port", 4244, "base server port")
	ipaddrPtr := flag.String("ip", "127.0.0.1", "hd base IP address (server or client)")

	flag.Parse()

	chanThrpt := make(chan map[string]float64)

	// Get operations to perform
	operations := strings.Split(*operationsPtr, " ")

	// Launch goroutines in a loop
	for nrinstances := 0; nrinstances < *nrinstancesPtr; nrinstances++ {
		port := *basePortPtr + nrinstances
		baseURL := "http://" + *ipaddrPtr + ":" + strconv.Itoa(port) + "/"
		if *typePtr == "client" {
			baseURL = baseURL + "proxy/arc/"
		}
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
		thrpt := <-chanThrpt
		log.Println("Instance ID", nrinstances, "Throughput=", thrpt, "Mo/s")
	}

	// Delete Traffic Control
	if *tcKindPtr != "" && *tcOptionsPtr != "" && *tcPortPtr != 0 {
		utils.DeleteTrafficRules()
	}
}
