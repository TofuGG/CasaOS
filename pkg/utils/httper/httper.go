package httper

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/IceWhaleTech/CasaOS/pkg/config"
	"github.com/tidwall/gjson"
)

// sharedHTTPClient is a shared HTTP client with a reusable transport to enable
// DNS caching and connection reuse across all HTTP calls in the application.
// This prevents excessive DNS lookups when many requests target the same host.
var sharedHTTPTransport = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

// newHTTPClient returns a client using the shared transport with DNS caching.
// Each caller can specify its own timeout while still benefiting from the
// shared connection pool and DNS cache.
func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: sharedHTTPTransport,
	}
}

// 发送GET请求
// url:请求地址
// response:请求返回的内容
func Get(url string, head map[string]string) (response string) {
	client := newHTTPClient(30 * time.Second)
	req, err := http.NewRequest("GET", url, nil)

	for k, v := range head {
		req.Header.Add(k, v)
	}
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		// 需要错误日志的处理
		// logger.Error(error)
		return ""
		// panic(error)
	}
	defer resp.Body.Close()
	var buffer [512]byte
	result := bytes.NewBuffer(nil)
	for {
		n, err := resp.Body.Read(buffer[0:])
		result.Write(buffer[0:n])
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			// logger.Error(err)
			return ""
			//	panic(err)
		}
	}
	response = result.String()
	return
}

// 发送GET请求
// url:请求地址
// response:请求返回的内容
func PersonGet(url string) (response string) {
	client := newHTTPClient(5 * time.Second)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		// 需要错误日志的处理
		// logger.Error(error)
		return ""
		// panic(error)
	}
	defer resp.Body.Close()
	var buffer [512]byte
	result := bytes.NewBuffer(nil)
	for {
		n, err := resp.Body.Read(buffer[0:])
		result.Write(buffer[0:n])
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			// logger.Error(err)
			return ""
			//	panic(err)
		}
	}
	response = result.String()
	return
}

// 发送POST请求
// url:请求地址，data:POST请求提交的数据,contentType:请求体格式，如：application/json
// content:请求放回的内容
func Post(url string, data []byte, contentType string, head map[string]string) (content string) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Add("content-type", contentType)
	for k, v := range head {
		req.Header.Add(k, v)
	}
	if err != nil {
		panic(err)
	}

	client := newHTTPClient(5 * time.Second)
	resp, error := client.Do(req)
	if error != nil {
		fmt.Println(error)
		return
	}
	defer resp.Body.Close()

	result, _ := io.ReadAll(resp.Body)
	content = string(result)
	return
}

// 发送POST请求
// url:请求地址，data:POST请求提交的数据,contentType:请求体格式，如：application/json
// content:请求放回的内容
func ZeroTierGet(url string, head map[string]string) (content string, code int) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	for k, v := range head {
		req.Header.Add(k, v)
	}
	if err != nil {
		panic(err)
	}

	client := newHTTPClient(20 * time.Second)
	resp, error := client.Do(req)

	if error != nil {
		panic(error)
	}
	defer resp.Body.Close()
	code = resp.StatusCode
	result, _ := io.ReadAll(resp.Body)
	content = string(result)
	return
}

// 发送GET请求
// url:请求地址
// response:请求返回的内容
func OasisGet(url string) (response string) {
	head := make(map[string]string)

	t := make(chan string)

	go func() {
		str := Get(config.ServerInfo.ServerApi+"/token", nil)

		t <- gjson.Get(str, "data").String()
	}()
	head["Authorization"] = <-t

	return Get(url, head)
}
