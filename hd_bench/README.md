# Hyperdrive bench
*hd_bench* repository provides an hyperdrive injector which aims at benchmarking both hyperdrive servers and client

# Getting started

## Compilation

```
>> cd hd_bench/src
>> go build -o injector
```

## Running the tests

```
>> cd hd_bench/src
>> go test
2019/04/25 10:23:45 type:server, req:put, key:key0, file:/etc/hosts, size:236, url:http://127.0.0.1:4244/
2019/04/25 10:23:45 type:server, req:get, key:key0, file:/etc/hosts, size:236, url:http://127.0.0.1:4244/
2019/04/25 10:23:45 type:server, req:del, key:key0, file:/etc/hosts, size:236, url:http://127.0.0.1:4244/
2019/04/25 10:23:45 type:client, req:put, key:key0, file:/etc/hosts, size:236, url:http://127.0.0.1:4244/
PASS
ok  	github.com/massiou/golang/hd_bench/src	0.002s
```

## Usage

```
Usage of ./injector:
  -hd-type string
    	Choose between hyperdrive 'server' or 'client' (default "server")
  -ip string
    	hd base IP address (server or client) (default "127.0.0.1")
  -nrinstances int
    	number of HD clients/servers (default 1)
  -nrkeys int
    	number of keys per goroutine (default 1)
  -operations string
    	worload operations 'put' or 'put get' or 'put del' or 'put get del' (default "put")
  -payload-files string
    	payload files (default "/etc/hosts /usr/bin/gdb")
  -port int
    	base server port (default 4244)
  -tc-kind string
    	traffic control kind
  -tc-opt string
    	traffic control options
  -tc-port int
    	traffic control port
  -w int
    	number of injector workers  (default 10)

```
