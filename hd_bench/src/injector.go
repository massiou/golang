package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
)

// Initialize arguments
var hdType = flag.String("hd-type", "server", "Choose between hyperdrive 'server' or 'client'")
var payloads = flag.String("payload-files", "/etc/hosts /usr/bin/gdb", "payload files")
var nrinstances = flag.Int("nrinstances", 1, "number of HD clients/servers")
var nrkeys = flag.Int("nrkeys", 1, "number of keys per goroutine")
var tcKind = flag.String("tc-kind", "", "traffic control kind")
var tcOptions = flag.String("tc-opt", "", "traffic control options")
var tcPort = flag.Int("tc-port", 0, "traffic control port")
var operations = flag.String("operations", "put", "worload operations 'put' or 'put get' or 'put del' or 'put get del'")
var basePort = flag.Int("port", 4244, "base server port")
var ipaddr = flag.String("ip", "127.0.0.1", "hd base IP address (server or client)")
var nrworkers = flag.Int("w", 10, "number of injector workers ")

// performWorkload
func performWorkload(
	hdType string, // server or client
	operations string, // PUT / GET / DELETE
	baseURL string,
	keys []string, // list of keys
	payloadFiles []string,
	size int,
	chanThrpt chan float64,
	wg *sync.WaitGroup) {

	client := CustomClient(100) // HTTP client
	opArray := strings.Split(operations, " ")

	var keysGenerated []string
	var totalSize int
	var errors int
	start := time.Now()

	for _, operation := range opArray {
		errors = 0
		// Loop on all keys
		for _, key := range keys {
			// Build request
			randIdx := rand.Int() % len(payloadFiles)
			payloadFile := payloadFiles[randIdx]
			hdReq := hdRequest{hdType, operation, key, payloadFile, size, baseURL}
			opRequest := OpKey(hdReq)
			res, err := client.Do(opRequest)

			if err != nil {
				log.Fatal("err=", err)
			}
			if res.StatusCode >= 300 {
				log.Println("status code=", res.StatusCode, "res=", res)
				errors++
			}

			// Compare PUT and GET answer
			if operation == "get" {
				comparison, resp := compareGetPut(payloadFile, res)
				if comparison == false {
					log.Println("GET response body different from original PUT, expected:", payloadFile)
					log.Fatal("Response content length=", len(resp))
				}
			} else {
				io.Copy(ioutil.Discard, res.Body)
			}
			// Close the current request
			res.Body.Close()

			// After a PUT on client hdproxy, get the generate key
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

	log.Println("nr errors=", errors)
}

func getThroughput(start time.Time, size int) float64 {
	var throughput float64
	elapsed := float64(time.Since(start)) / math.Pow10(9)

	if elapsed != 0 {
		// in Mo/s
		throughput = float64(size) / elapsed
		glog.V(2).Info("Throughput: ", throughput, " Mo/s")
	}
	log.Println(throughput)
	return throughput
}

func compareGetPut(file string, res *http.Response) (bool, string) {
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
	strResp := string(responseData)
	// Return comparison status
	return strResp == strData, strResp
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
	payloadFiles []string,
	wgMain *sync.WaitGroup,
	chanThrpt chan float64,
	workers int) {

	var wgWorkload sync.WaitGroup
	//size := getFileSize(payloadFile)
	size := 0
	for i := 0; i < workers; i++ {
		// generate nrkeys random keys
		keys := GenerateKeys(hdType, nrkeys)

		wgWorkload.Add(1)
		go performWorkload(hdType, operations, baseserver, keys, payloadFiles, size, chanThrpt, &wgWorkload)
	}

	wgWorkload.Wait()
	wgMain.Done()
}

func main() {
	var wgMain sync.WaitGroup
	chanThrpt := make(chan float64)
	//start := time.Now()

	// Parse command-line arguments
	flag.Parse()

	files := strings.Split(*payloads, " ")

	// Launch goroutines in a loop
	for nri := 0; nri < *nrinstances; nri++ {
		port := *basePort + nri
		baseURL := "http://" + *ipaddr + ":" + strconv.Itoa(port) + "/"
		wgMain.Add(1)
		go mainFunc(*hdType, *operations, baseURL, *nrkeys, files, &wgMain, chanThrpt, *nrworkers)
	}
	go func() {
		defer close(chanThrpt)
		wgMain.Wait()
	}()

	// Launch Traffic Control
	if *tcKind != "" && *tcOptions != "" && *tcPort != 0 {
		TrafficControl(*tcKind, *tcOptions, *tcPort)
	}
	idx := 0
	for thr := range chanThrpt {
		log.Println("worker", idx, "throughput=", thr/math.Pow10(6), "Mo/s")
		idx++
	}

	//totalSize := (*nrworkers) * (*nrkeys) * getFileSize(*payload)
	//finalThr := getThroughput(start, totalSize)

	//log.Println("Total throughput:", finalThr/math.Pow10(6), "Mo/s")

	// Delete Traffic Control
	if *tcKind != "" && *tcOptions != "" && *tcPort != 0 {
		DeleteTrafficRules()
	}
}
