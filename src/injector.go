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
	BaseClient         = "http://127.0.0.1:8889/"
	PortClient         = 8889
	PortServer         = 4244
	maxFileDescriptors = 1000
)

// performPutGet
func performPutGet(hdType string, baseURL string, nrkeys int, payloadFile string, wg *sync.WaitGroup, throughputChan chan float64) {

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

	// defer wait group done
	defer log.Println("End of performPutGet ", number, baseURL)

	key := "defaultKey"
	log.Println(hdType)
	for elt := 0; elt < nrkeys; elt++ {
		if hdType == "server" {
			key = utils.GenerateKey(64)
		} else if hdType == "client" {
			key = fmt.Sprintf("dir-%d/obj-%d", number, elt)
		}

		// Build PUT request
		log.Println("Put key: ", key, "on", baseURL)
		putRequest := utils.PutKey(hdType, key, payloadFile, size, baseURL)

		res, err := client.Do(putRequest)

		if err != nil {
			log.Fatal(err)
		}

		if res.StatusCode != 204 && res.StatusCode != 200 {
			log.Fatal(res.StatusCode)
		}

		res.Body.Close()

		totalSize += int(size)

		elapsed := int(time.Since(start)) / int(math.Pow10(9))

		if elapsed != 0 {
			throughput = float64((totalSize / elapsed) / int(math.Pow10(6)))

			fmt.Println("totalSize=", totalSize, "elapsed=", elapsed)
			fmt.Println("Throughput: ", throughput, "Mo/s")
		}

		/*
			// Build GET request
			getRequest := utils.GetKey(key, baseURL)
			log.Println("Get key: ", key)
			res2, err2 := client.Do(getRequest)

			if res2.StatusCode != 200 {
				log.Fatal(err2)
			}
			res2.Body.Close()
		*/
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

func mainFunc(hdType string, baseserver string, nrroutines int, nrkeys int, payloadFile string, wgMain *sync.WaitGroup) {
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
		go performPutGet(hdType, baseserver, nrkeys, payloadFile, &wg, throughputChan)
		throughput := <-throughputChan
		thrSlice = append(thrSlice, throughput)
		fmt.Println("Routine", i, "Throughput=", throughput, "Mo/s")
	}

	wg.Wait()

	end := time.Now().Unix()

	log.Println(int(end) - int(start))

	fmt.Println("Routines: ", thrSlice)
}

var wgMain sync.WaitGroup

func main() {

	// Arguments
	workersPtr := flag.Int("workers", 64, "number of workers in parallel")
	typePtr := flag.String("hd-type", "server", "Choose between hyperdrive 'server' or 'client'")
	payloadPtr := flag.String("payload-file", "/etc/hosts", "payload file")
	nrinstancesPtr := flag.Int("nrinstances", 1, "number of HD clients/servers")
	nrkeysPtr := flag.Int("nrkeys", 1, "number of keys per goroutine")

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
		go mainFunc(*typePtr, baseURL, *workersPtr, *nrkeysPtr, *payloadPtr, &wgMain)

	}
	wgMain.Wait()
}
