package main

import (
	"bytes"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/kataras/golog"
)

func main() {
	start := time.Now()
	http.HandleFunc("/testconn", func(writer http.ResponseWriter, request *http.Request) {
		// 禁用连接复用，每次请求后关闭连接
		needClose := request.URL.Query().Get("close") == "1"
		if needClose {
			writer.Header().Set("Connection", "close")
		}

		defer request.Body.Close()
		data, err := io.ReadAll(request.Body)
		if err != nil {
			golog.Errorf("readbody %s", err)
			return
		}
		n, err := writer.Write(data)
		if err != nil {
			golog.Errorf("write err %s", err)
		}
		if n != len(data) {
			golog.Errorf("write not equal, expected %d, got %d", len(data), n)
		}
	})
	go http.ListenAndServe("127.0.0.1:9977", nil)
	time.Sleep(time.Second)
	runReq()
	golog.Infof("total time: %.2f", time.Since(start).Seconds())
}

func runReq() {
	proxy, _ := url.Parse("socks5://127.0.0.1:1111")
	var wg sync.WaitGroup
	// var connDone atomic.Uint32
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// defer func() {
			// 	golog.Infof("done %d", connDone.Add(1))
			// }()
			client := http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxy)}, Timeout: time.Second * 5}
			for j := 0; j < 30; j++ {
				data := randBytes()
				u := "http://127.0.0.1:9977/testconn"
				if rand.Int()%2 == 0 {
					u = "http://127.0.0.1:9977/testconn?close=1"
				}
				resp, err := client.Post(u, "application/octet-stream", bytes.NewReader(data))
				if err != nil {
					golog.Error(err)
					return
				}
				newData, err := io.ReadAll(resp.Body)
				if err != nil {
					golog.Error(err)
					return
				}
				_ = resp.Body.Close()
				if !bytes.Equal(data, newData) {
					golog.Error("data not equal")
					return
				}
			}
		}()
	}
	wg.Wait()
}

func randBytes() []byte {
	randCount := rand.Intn(40960)
	data := make([]byte, randCount)
	rand.Read(data)
	return data
}
