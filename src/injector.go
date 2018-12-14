package main

import (
	"encoding/json"
	"flag"
	"fmt"
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

// performPutGet
func performPutGet(hdType string, operations []string, baseURL string, nrkeys int, payloadFile string, wg *sync.WaitGroup, throughputChan chan float64) {

	defer wg.Done()
	client := &http.Client{}

	throughput := 0.0
	var totalSize int
	start := time.Now()

	// Payload size is needed
	fi, errSize := os.Stat(payloadFile)

	if errSize != nil {
		log.Fatal("os.Stat() of", payloadFile, "error:", errSize)
	}

	// get the size
	size := fi.Size()

	// Store a random number to identify the current instance
	rand.Seed(time.Now().UTC().UnixNano())
	number := rand.Intn(1000)

	key := "defaultKey"

	// defer wait group done
	defer log.Println("End of performPutGetDel ", number, baseURL)

	for elt := 0; elt < nrkeys; elt++ {
		if hdType == "server" {
			key = utils.GenerateKey(64)
		} else if hdType == "client" {
			key = fmt.Sprintf("dir-%d/obj-%d", number, elt)
		}

		for _, operation := range operations {

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

			res.Body.Close()

			// After a PUT on client hdproxy, get the generated key
			if operation == "put" && hdType == "client" {
				randomKey := res.Header.Get("Scal-Key")
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

				res2.Body.Close()
			}

		}
		// Update total put size
		totalSize += int(size)

		// Get elapsed time and convert it from nano to seconds
		elapsed := int(time.Since(start)) / int(math.Pow10(9))

		if elapsed != 0 {
			// in Mo/s
			throughput = float64((totalSize / elapsed) / int(math.Pow10(6)))

			fmt.Println("totalSize=", totalSize, "nrkeys=", nrkeys, "elapsed=", elapsed)
			fmt.Println("Throughput: ", throughput, "Mo/s")
		}
	}
	throughputChan <- throughput

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

// performDelClient hyperdrive client
func PerformDelClient(client *http.Client, baseclient string, nrkeys int, wg *sync.WaitGroup) bool {
	/* XXX Should be done before calling goroutine */
	wg.Add(1)
	fmt.Println("OK")

	return true

}

func mainFunc(hdType string, operations []string, baseserver string, nrroutines int, nrkeys int, payloadFile string, wgMain *sync.WaitGroup) {
	defer wgMain.Done()
	log.Println("Launch injector routines: ", nrroutines)

	// Create wait group object
	var wg sync.WaitGroup

	throughputChan := make(chan float64)

	var thrSlice []float64

	start := time.Now().Unix()
	// Perform PUT & GET concurrently
	for i := 0; i < nrroutines; i++ {
		wg.Add(1)
		go performPutGet(hdType, operations, baseserver, nrkeys, payloadFile, &wg, throughputChan)
		throughput := <-throughputChan
		thrSlice = append(thrSlice, throughput)
		fmt.Println("Routine", i, "Throughput=", throughput, "Mo/s")
	}

	wg.Wait()

	end := time.Now().Unix()

	log.Println(int(end) - int(start))

	fmt.Println("Operations=", operations, "Throughput=", thrSlice)
}

var wgMain sync.WaitGroup

func main() {

	// Arguments
	workersPtr := flag.Int("workers", 64, "number of workers in parallel")
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
		go mainFunc(*typePtr, operations, baseURL, *workersPtr, *nrkeysPtr, *payloadPtr, &wgMain)

	}
	wgMain.Wait()
}
