package main

import (
	"flag"
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
	operations string, // PUT / GET / DELETE
	baseURL string,
	keys []string, // list of keys
	payloadFile string,
	size int,
	chanThrpt chan float64,
	wg *sync.WaitGroup) {

	client := utils.CustomClient(100) // HTTP client
	opArray := strings.Split(operations, " ")

	var keysGenerated []string
	var totalSize int

	start := time.Now()

	for _, operation := range opArray {
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
				gKey := res.Header.Get("Scal-Key")
				keysGenerated = append(keysGenerated, gKey)
			}

			// Update total put size
			totalSize += int(size)
		}
		if operation == "put" && hdType == "client" {
			keys = keysGenerated
		}

	}
	wg.Done()

	chanThrpt <- getThroughput(start, totalSize)
}

func getThroughput(start time.Time, size int) float64 {
	var throughput float64
	elapsed := float64(time.Since(start)) / math.Pow10(9)

	log.Println(elapsed)
	log.Println(size)

	if elapsed != 0 {
		// in Mo/s
		throughput = float64(size) / elapsed
		glog.V(2).Info("Throughput: ", throughput, " Mo/s")
	}
	log.Println(throughput)
	return throughput
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

func getFileSize(path string) int {
	// Payload size is needed for PUT header and to compute throughput
	fi, errSize := os.Stat(path)

	if errSize != nil {
		log.Fatal("os.Stat() of", path, "error:", errSize)
	}
	return int(fi.Size())
}

func mainFunc(
	hdType string,
	operations string,
	baseserver string,
	nrkeys int,
	payloadFile string,
	wgMain *sync.WaitGroup,
	chanThrpt chan float64,
	workers int) {

	var wgWorkload sync.WaitGroup
	size := getFileSize(payloadFile)

	for i := 0; i < workers; i++ {
		// generate nrkeys random keys
		keys := utils.GenerateKeys(hdType, nrkeys)

		wgWorkload.Add(1)
		go performWorkload(hdType, operations, baseserver, keys, payloadFile, size, chanThrpt, &wgWorkload)
	}

	wgWorkload.Wait()
	wgMain.Done()
}

func main() {
	var wgMain sync.WaitGroup
	// Arguments
	typePtr := flag.String("hd-type", "server", "Choose between hyperdrive 'server' or 'client'")
	payloadPtr := flag.String("payload-file", "/etc/hosts", "payload file")
	nrinstancesPtr := flag.Int("nrinstances", 1, "number of HD clients/servers")
	nrkeysPtr := flag.Int("nrkeys", 1, "number of keys per goroutine")
	tcKindPtr := flag.String("tc-kind", "", "traffic control kind")
	tcOptionsPtr := flag.String("tc-opt", "", "traffic control options")
	tcPortPtr := flag.Int("tc-port", 0, "traffic control port")
	operationsPtr := flag.String("operations", "put", "worload operations 'put' or 'put get' or 'put del' or 'put get del'")
	basePortPtr := flag.Int("port", 4244, "base server port")
	ipaddrPtr := flag.String("ip", "127.0.0.1", "hd base IP address (server or client)")
	nrworkersPtr := flag.Int("w", 10, "number of injector workers ")

	flag.Parse()

	chanThrpt := make(chan float64)

	start := time.Now()

	// Launch goroutines in a loop
	for nrinstances := 0; nrinstances < *nrinstancesPtr; nrinstances++ {
		port := *basePortPtr + nrinstances
		baseURL := "http://" + *ipaddrPtr + ":" + strconv.Itoa(port) + "/"
		wgMain.Add(1)
		go mainFunc(*typePtr, *operationsPtr, baseURL, *nrkeysPtr, *payloadPtr, &wgMain, chanThrpt, *nrworkersPtr)

	}

	go func() {
		defer close(chanThrpt)
		wgMain.Wait()
	}()

	// Launch Traffic Control
	if *tcKindPtr != "" && *tcOptionsPtr != "" && *tcPortPtr != 0 {
		utils.TrafficControl(*tcKindPtr, *tcOptionsPtr, *tcPortPtr)
	}
	idx := 0
	for thr := range chanThrpt {
		log.Println("worker", idx, "throughput=", thr/math.Pow10(6), "Mo/s")
		idx++
	}

	totalSize := (*nrworkersPtr) * (*nrkeysPtr) * getFileSize(*payloadPtr)
	finalThr := getThroughput(start, totalSize)

	log.Println("Total throughput:", finalThr/math.Pow10(6), "Mo/s")

	// Delete Traffic Control
	if *tcKindPtr != "" && *tcOptionsPtr != "" && *tcPortPtr != 0 {
		utils.DeleteTrafficRules()
	}
}
