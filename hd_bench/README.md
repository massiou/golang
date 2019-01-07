# Hyperdrive bench
*hd_bench* repository provides an hyperdrive injector which aims at benchmarking both hyperdrive servers and client

# Getting started

## Compilation

```
>> cd hd_bench/src
>> go build injector.go
```

## Running the tests

```
>> cd hd_bench/src
>> go test
2019/01/07 11:02:57 put  key:  key0 on http://127.0.0.1:4244/`
2019/01/07 11:02:57 keys= [key0] throughput= 0
2019/01/07 11:02:57 get  key:  key0 on http://127.0.0.1:4244/
2019/01/07 11:02:57 keys= [key0] throughput= 0
2019/01/07 11:02:57 del  key:  key0 on http://127.0.0.1:4244/
2019/01/07 11:02:57 keys= [key0] throughput= 0
PASS
ok  	_/home/mvelay/workspace/github/golang/hd_bench/src	0.023s
```

## Usage

```
>> cd hd_bench/src
>> ./injector -h
Usage of ./injector:
  -hd-type string
    	Choose between hyperdrive 'server' or 'client' (default "server")
  -nrinstances int
    	number of HD clients/servers (default 1)
  -nrkeys int
    	number of keys per goroutine (default 1)
  -payload-file string
    	payload file (default "/etc/hosts")
```
