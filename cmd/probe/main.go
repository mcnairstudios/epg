package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gavinmcnair/epg/internal/httpclient"
)

func main() {
	insecure := httpclient.InsecureBrowser()

	// siba.com.co is HTTP not HTTPS - use regular client
	fmt.Println("=== siba.com.co (regular http client) ===")
	form := url.Values{}
	form.Set("servicio", "10")
	form.Set("ini", fmt.Sprint(time.Now().Truncate(24*time.Hour).Unix()))
	form.Set("end", fmt.Sprint(time.Now().Truncate(24*time.Hour).Add(24*time.Hour).Unix()))
	form.Set("chn", "554")
	req, _ := http.NewRequest(http.MethodPost, "http://devportal.siba.com.co/index.php?action=grilla", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("ERROR (default): %v\n", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("Status: %d, Body (first 300): %s\n", resp.StatusCode, truncate(body, 300))
	}

	// directv.com.uy - try with different cookie/header combos
	fmt.Println("\n=== directv.com.uy - attempt with more headers ===")
	now := time.Now()
	payload := fmt.Sprintf(`{"filterParameters":{"day":%d,"time":0,"minute":0,"month":%d,"year":%d,"offSetValue":0,"filtersScreenFilters":[""],"isHd":"","isChannelDetails":"Y","channelNum":"1541","channelName":"MAGAHD"}}`, now.Day(), int(now.Month()), now.Year())
	req2, _ := http.NewRequest(http.MethodPost, "https://www.directv.com.uy/guia/ChannelDetail.aspx/GetProgramming", bytes.NewReader([]byte(payload)))
	req2.Header.Set("Accept", "*/*")
	req2.Header.Set("Accept-Language", "es-419,es;q=0.9")
	req2.Header.Set("Connection", "keep-alive")
	req2.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req2.Header.Set("Cookie", "PGCSS=16384; PGLang=S; PGCulture=es-UY;")
	req2.Header.Set("Origin", "https://www.directv.com.uy")
	req2.Header.Set("Referer", "https://www.directv.com.uy/guia/ChannelDetail.aspx?id=1541&name=MAGAHD")
	req2.Header.Set("Sec-Fetch-Dest", "empty")
	req2.Header.Set("Sec-Fetch-Mode", "cors")
	req2.Header.Set("Sec-Fetch-Site", "same-origin")
	req2.Header.Set("X-Requested-With", "XMLHttpRequest")
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")
	req2.Header.Set("sec-ch-ua", `"Not;A=Brand";v="99", "Brave";v="139", "Chromium";v="139"`)
	req2.Header.Set("sec-ch-ua-mobile", "?0")
	req2.Header.Set("sec-ch-ua-platform", `"Windows"`)
	resp2, err := insecure.Do(req2)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		body, _ := io.ReadAll(resp2.Body)
		resp2.Body.Close()
		fmt.Printf("Status: %d\nHeaders: %v\nBody: %s\n", resp2.StatusCode, resp2.Header, truncate(body, 500))
	}
}

func truncate(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n])
	}
	return string(b)
}
