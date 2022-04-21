package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
)

var um int = 0
var bodyMap = make(map[string][]byte)
var headMap = make(map[string]http.Header)
var lock sync.Mutex

func main() {
	var host string
	var listen uint
	flag.StringVar(&host, "url", "http://127.0.0.1", "web server url")
	flag.UintVar(&listen, "port", 8888, "localhost listen port")
	flag.Parse()

	log.Printf("[service start] http://127.0.0.1:%d", listen)
	http.Handle("/", CacheHandle{Host: host, BMap: bodyMap, HMap: headMap})
	addr := ":" + strconv.Itoa(int(listen))
	log.Fatal(http.ListenAndServe(addr, nil))
}

type CacheHandle struct {
	Host string
	BMap map[string][]byte
	HMap map[string]http.Header
}

func (h CacheHandle) ServeHTTP(w http.ResponseWriter, r *http.Request)  {
	log.Printf("[Request] %s %s %s", r.Method, r.URL.String(), r.Proto)

	uri := r.RequestURI

	lock.Lock()
	bodyCache, ok1 := h.BMap[uri]
	headCache, ok2 := h.HMap[uri]
	lock.Unlock()

	if ok1 && ok2 {
		for k, vv := range headCache {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}

		k, _ := w.Write(bodyCache)
		log.Printf("[Response from cache] %s %d bytes", uri, k)
	} else {
		url := h.Host + uri
		log.Printf("[Request real server] %s", uri)

		resp, err := http.Get(url)
		if err != nil {
			log.Panicln(err)
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Panicln(err)
		}

		lock.Lock()
		h.BMap[uri] = body
		h.HMap[uri] = resp.Header.Clone()
		um += len(body)
		lock.Unlock()

		log.Println("--------------Response Headers--------------")
		log.Printf("%s %s", resp.Proto, resp.Status)
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
				log.Println(k,": ", v)
			}
		}
		log.Println("--------------------------------------------")

		k, _ := w.Write(body)
		log.Printf("[Response from real server] %s %d bytes", uri, k)
	}

	log.Printf("[CACHE SIZE %.2fM]", float64(um) / 1048576)
}
