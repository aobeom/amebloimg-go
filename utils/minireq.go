package utils

import (
    "io"
    "io/ioutil"
    "log"
    "net"
    "net/http"
    "time"

    "golang.org/x/net/proxy"
)

func s5Proxy(proxyURL string) (transport *http.Transport) {
    dialer, err := proxy.SOCKS5("tcp", proxyURL,
        nil,
        &net.Dialer{
            Timeout:   30 * time.Second,
            KeepAlive: 30 * time.Second,
        },
    )
    if err != nil {
        log.Fatal(" [S5 Proxy Error]: ", err)
    }
    transport = &http.Transport{
        Proxy:               nil,
        Dial:                dialer.Dial,
        TLSHandshakeTimeout: 10 * time.Second,
    }
    return
}

// HTTPClient 设置 http 请求
func HTTPClient(proxy string) (client http.Client) {
    client = http.Client{Timeout: 30 * time.Second}
    if proxy != "" {
        transport := s5Proxy(proxy)
        client = http.Client{Timeout: 30 * time.Second, Transport: transport}
    }
    return
}

// Get get 请求
func Get(url string, headers map[string][]string, params map[string]string) []byte {
    httpClient := http.Client{Timeout: 30 * time.Second}
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        log.Println(" [Get - Request Error]: ", err)
    }
    req.Header = headers
    if params != nil {
        q := req.URL.Query()
        for k, v := range params {
            q.Add(k, v)
        }
        req.URL.RawQuery = q.Encode()
    }

    res, err := httpClient.Do(req)

    if err != nil {
        log.Println(" [Get - Response Error]: ", err)
    }

    if res.StatusCode != 200 {
        log.Println(" [Get - Response Code != 200]: ", err)
    }

    body, err := ioutil.ReadAll(res.Body)
    if err != nil {
        log.Println(" [Get - Body Error]: ", err)
    }
    return body
}

// Post post 请求
func Post(url string, headers map[string][]string, reader io.Reader) []byte {
    httpClient := http.Client{Timeout: 30 * time.Second}
    req, err := http.NewRequest("POST", url, reader)
    if err != nil {
        log.Println(" [Post - Request Error]: ", err)
    }

    req.Header = headers
    res, err := httpClient.Do(req)

    if err != nil {
        log.Println(" [Post - Response Error]: ", err)
    }
    body, err := ioutil.ReadAll(res.Body)
    if err != nil {
        log.Println(" [Get - Body Error]: ", err)
    }
    return body
}
