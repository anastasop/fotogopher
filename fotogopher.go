package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/nfnt/resize"
)

type SnapshotRequest struct {
	url    string
	width  int
	height int

	done chan *SnapshotResult
}

type SnapshotResult struct {
	img   *bytes.Buffer
	error error
}

var addr = flag.String("a", ":8080", "addr:port for the server")
var nworkers = flag.Int("n", 0, "number of workers to spawn, 0 for unlimited")
var timeout = flag.Duration("t", time.Duration(45)*time.Second, "timeout for the screenshot")

var workQueue = make(chan *SnapshotRequest)

func takeSnapshot(req *SnapshotRequest) (res *SnapshotResult) {
	defer func() {
		if r := recover(); r != nil {
			res = &SnapshotResult{nil, fmt.Errorf("snapshot failed because of a panic: %v", r)}
		}
	}()

	// TODO handle timeout (exit code 164) differently
	body, err := exec.Command("timeout",
		strconv.FormatInt(int64(timeout.Seconds()), 10)+"s",
		"phantomjs", "capture.js", req.url).Output()
	if err != nil {
		return &SnapshotResult{nil, fmt.Errorf("phantom.js failed: %s", err)}
	}

	dbuf := make([]byte, base64.StdEncoding.DecodedLen(len(body)))
	n, err := base64.StdEncoding.Decode(dbuf, []byte(body))
	if err != nil {
		return &SnapshotResult{nil, fmt.Errorf("base64 decoding failed: %s", err)}
	}

	img, err := jpeg.Decode(bytes.NewBuffer(dbuf[:n]))
	if err != nil {
		return &SnapshotResult{nil, fmt.Errorf("jpeg decoding failed: %s", err)}
	}

	scaled := resize.Resize(uint(req.width), uint(req.height), img, resize.Bilinear)

	var b bytes.Buffer
	if err := jpeg.Encode(&b, scaled, nil); err != nil {
		return &SnapshotResult{nil, fmt.Errorf("jpeg encoding failed: %s", err)}
	}

	return &SnapshotResult{&b, nil}
}

func takeSnapshotContext(req *SnapshotRequest) {
	log.Printf("[%p] take snapshot for %s", req, req.url)
	res := takeSnapshot(req)
	if res.error != nil {
		log.Printf("[%p] take snapshot for %s failed: ", req, res.error)
	} else {
		log.Printf("[%p] take snapshot for %s completed", req, req.url)
	}
	req.done <- res
}

func pooledWorker(work <-chan *SnapshotRequest) {
	for {
		takeSnapshotContext(<-work)
	}
}

func freeWorker(work <-chan *SnapshotRequest) {
	for {
		go takeSnapshotContext(<-work)
	}
}

func snapshotHandler(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()

	url := req.Form.Get("url")
	if url == "" {
		http.Error(w, "Request does not contain a url", http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	width := 0
	if s := req.Form.Get("width"); s != "" {
		if i, err := strconv.Atoi(s); err == nil {
			width = i
		}
	}

	height := 0
	if s := req.Form.Get("height"); s != "" {
		if i, err := strconv.Atoi(s); err == nil {
			height = i
		}
	}

	if width == 0 && height == 0 {
		http.Error(w, "Request has both is dimensions zero", http.StatusBadRequest)
		return
	}

	log.Printf("accepted snapshot request for %s", url)

	done := make(chan *SnapshotResult)
	// in case we have a pool of workers don't wait too much
	select {
	case workQueue <- &SnapshotRequest{url: url, width: width, height: height, done: done}:
		// nothing to do here
	case <-time.After(time.Duration(10) * time.Second):
		http.Error(w, "Please try again later, we are too busy now", http.StatusServiceUnavailable)
	}

	// phantom.js runs under timeout but add a timeout here just in case
	select {
	case res := <-done:
		if res.error != nil {
			http.Error(w, "Failed to render: "+res.error.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		io.Copy(w, res.img)

	case <-time.After(*timeout):
		http.Error(w, "Request for "+url+" timed out", http.StatusGatewayTimeout)
	}
}

const help_page = `<!doctype html>
<html>
  <head><title>Snapshot</title></head>
  <body>
    <h1>Snapshot</h1>
    This is a web service that takes snapshots of web sites. API only. Endpoints are like:
<pre>
<a href="/snapshot?width=640&height=480&url=http://www.skroutz.gr">/snapshot?width=640&height=480&url=http://www.skroutz.gr</a>
</pre>

If width or height is 0 then it will be set to an aspect ration preserving value.
  </body>
</html>
`

func main() {
	flag.Parse()

	if *nworkers > 0 {
		for i := 0; i < *nworkers; i++ {
			go pooledWorker(workQueue)
		}
	} else {
		go freeWorker(workQueue)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, help_page)
	})
	http.HandleFunc("/snapshot", snapshotHandler)
	log.Print("Server started")
	log.Fatal(http.ListenAndServe(*addr, nil))
}
