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

  -alsologtostderr
    	log to standard error as well as files
  -hd-type string
    	Choose between hyperdrive 'server' or 'client' (default "server")
  -ip string
    	hd base IP address (server or client) (default "127.0.0.1")
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace
  -log_dir string
    	If non-empty, write log files in this directory
  -logtostderr
    	log to standard error instead of files
  -nrinstances int
    	number of HD clients/servers (default 1)
  -nrkeys int
    	number of keys per goroutine (default 1)
  -operations string
    	worload operations 'put' or 'put get' or 'put del' or 'put get del' (default "put")
  -payload-file string
    	payload file (default "/etc/hosts")
  -port int
    	base server port (default 4244)
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -tc-kind string
    	traffic control kind
  -tc-opt string
    	traffic control options
  -tc-port int
    	traffic control port
  -v value
    	log level for V logs
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
  -w int
    	number of injector workers  (default 10)

```
