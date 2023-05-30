package main

import (
	"chat/config"
	"chat/handler"
	"chat/repo"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	target    = "https://api.chatai.beauty" // 目标域名
	httpProxy = "http://127.0.0.1:10809"    // 本地代理地址和端口
)

func main() {
	http.HandleFunc("/", handleRequest)
	http.ListenAndServe(":9000", nil)
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// 过滤无效URL
	_, err := url.Parse(r.URL.String())
	if err != nil {
		log.Println("Error parsing URL: ", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	log.Printf("[%s] %s", r.Method, r.RemoteAddr)
	c, err := config.Parse()
	if err != nil {
		log.Fatal(err)
	}
	sqliteRepo, err := repo.NewSQLiteRepo(c.DBName, c.InitUsers)
	if err != nil {
		log.Fatal(err)
	}
	p := handler.NewProxyHandler(c.OpenAIKey, sqliteRepo.User)

	auth := r.Header.Get("Authorization")

	log.Println("auth: ", auth)
	if auth == "" {
		w.WriteHeader(401)
		w.Write([]byte("Unauthorized"))
		return
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	if !p.CheckToken(token) {
		w.WriteHeader(401)
		w.Write([]byte("Unauthorized"))
		return
	}

	// 去掉环境前缀（针对腾讯云，如果包含的话，目前我只用到了test和release）
	newPath := strings.Replace(r.URL.Path, "/release", "", 1)
	newPath = strings.Replace(newPath, "/test", "", 1)

	// 拼接目标URL
	targetURL := target + newPath

	// 创建代理HTTP请求
	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		log.Println("Error creating proxy request: ", err.Error())
		http.Error(w, "Error creating proxy request", http.StatusInternalServerError)
		return
	}

	// 将原始请求头复制到新请求中
	for headerKey, headerValues := range r.Header {
		for _, headerValue := range headerValues {
			proxyReq.Header.Add(headerKey, headerValue)
		}
	}

	proxyReq.Header.Set("Authorization", "Bearer "+p.OpenAIKey)

	// 默认超时时间设置为60s
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// 本地测试通过代理请求 OpenAI 接口
	if os.Getenv("ENV") == "local" {
		proxyURL, _ := url.Parse(httpProxy) // 本地HTTP代理配置
		client.Transport = &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// 向 OpenAI 发起代理请求
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Println("Error sending proxy request: ", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// 将响应头复制到代理响应头中
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 将响应状态码设置为原始响应状态码
	w.WriteHeader(resp.StatusCode)

	// 将响应实体写入到响应流中（支持流式响应）
	var respBody []byte
	buf := make([]byte, 1024)
	for {
		if n, err := resp.Body.Read(buf); err == io.EOF || n == 0 {
			log.Println(string(respBody))
			return
		} else if err != nil {
			log.Println("error while reading respbody: ", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			if _, err = w.Write(buf[:n]); err != nil {
				log.Println("error while writing resp: ", err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			respBody = append(respBody, buf[:n]...)
			w.(http.Flusher).Flush()
		}
	}
}