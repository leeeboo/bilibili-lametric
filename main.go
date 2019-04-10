package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type ApiStat struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	TTL     int    `json:"ttl"`
}

type StatResp struct {
	ApiStat
	Data Stat `json:"data"`
}

type Stat struct {
	Mid       int64 `json:"mid"`
	Following int64 `json:"following"`
	Whisper   int64 `json:"whisper"`
	Black     int64 `json:"black"`
	Follower  int64 `json:"follower"`
}

type UpStatResp struct {
	ApiStat
	Data UpStat `json:"data"`
}

type UpStat struct {
	Archive ViewCount `json:"archive"`
	Article ViewCount `json:"article"`
}

type ViewCount struct {
	View int64 `json:"view"`
}

type ApiResponse struct {
	ApiError
	Data interface{} `json:"data,omitempty"`
}

type ApiError struct {
	ErrCode int    `json:"err_code"`
	ErrMsg  string `json:"err_msg"`
}

func apiSendError(errCode int, errMsg string, w http.ResponseWriter) {
	var res ApiResponse

	res.ErrCode = errCode
	res.ErrMsg = errMsg

	j, _ := json.Marshal(res)

	w.Header().Set("Content-Type", "application/json")

	w.Write(j)

	return
}

func apiSendData(data interface{}, w http.ResponseWriter) {

	j, _ := json.Marshal(data)

	w.Header().Set("Content-Type", "application/json")

	w.Write(j)

	return
}

type LaMetricData struct {
	Text string `json:"text"`
	Icon string `json:"icon"`
}

func handler(w http.ResponseWriter, r *http.Request) {

	mid := r.URL.Query().Get("mid")
	mid = strings.TrimSpace(mid)

	if mid == "" {
		log.Println("mid empty")
		apiSendError(-1, "mid empty", w)
		return
	}

	api := fmt.Sprintf("https://api.bilibili.com/x/relation/stat?vmid=%s&jsonp=jsonp", mid)

	data, err := httpGet(api, nil)

	if err != nil {
		log.Println(err)
		apiSendError(-1, err.Error(), w)
		return
	}

	var statResp StatResp

	err = json.Unmarshal(data, &statResp)

	if err != nil {
		log.Println(err)
		apiSendError(-1, err.Error(), w)
		return
	}

	if statResp.Code != 0 {
		log.Println(string(data))
		apiSendError(-1, err.Error(), w)
		return
	}

	//
	api = fmt.Sprintf("https://api.bilibili.com/x/space/upstat?mid=%s&jsonp=jsonp", mid)

	data, err = httpGet(api, nil)

	if err != nil {
		log.Println(err)
		apiSendError(-1, err.Error(), w)
		return
	}

	var upStatResp UpStatResp

	err = json.Unmarshal(data, &upStatResp)

	if err != nil {
		log.Println(err)
		apiSendError(-1, err.Error(), w)
		return
	}

	if upStatResp.Code != 0 {
		log.Println(string(data))
		apiSendError(-1, err.Error(), w)
		return
	}

	log.Println(upStatResp, statResp)

	resp := map[string]interface{}{
		"frames": []LaMetricData{
			{
				Text: fmt.Sprintf("%d", statResp.Data.Follower),
				Icon: "a61",
			},
			{
				Text: fmt.Sprintf("%d", upStatResp.Data.Archive.View),
				Icon: "a2361",
			},
			{
				Text: "3000",
				Icon: "i15732",
			},
		},
	}

	apiSendData(resp, w)
	return
}

func main() {
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func httpPost(api string, param map[string]interface{}) ([]byte, error) {

	buf := new(bytes.Buffer)

	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)

	err := enc.Encode(param)

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	resp, err := http.Post(api, "application/json", strings.NewReader(buf.String()))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	return body, err
}

func httpGet(api string, param map[string]interface{}) ([]byte, error) {

	queryStr, err := build(param)

	if err != nil {
		return nil, err
	}

	apiInfo, err := url.Parse(api)

	if err != nil {
		return nil, err
	}

	if apiInfo.RawQuery == "" {
		api = fmt.Sprintf("%s?%s", api, queryStr)
	} else {
		api = fmt.Sprintf("%s&%s", api, queryStr)
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	resp, err := http.Get(api)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	return body, err
}

func build(raw map[string]interface{}) (string, error) {

	p := make(map[string]string, 0)

	for k, v := range raw {

		switch vv := v.(type) {
		case []interface{}:

			parseNormal(p, vv, []string{k})

			break
		case map[string]interface{}:

			parseKeyValue(p, vv, []string{k})

			break
		default:

			p[k] = fmt.Sprintf("%s", vv)

			break
		}
	}

	data := url.Values{}

	for k, v := range p {
		data.Add(k, v)
	}

	return data.Encode(), nil
}

func parseKeyValue(p map[string]string, raw map[string]interface{}, keys []string) {

	for k, v := range raw {
		switch vv := v.(type) {
		case []interface{}:

			tmpKeys := append(keys, k)

			parseNormal(p, vv, tmpKeys)

			break
		case map[string]interface{}:

			tmpKeys := append(keys, k)

			parseKeyValue(p, vv, tmpKeys)

			break
		default:

			//keys = append(keys, k)

			var tmp []string

			for m, n := range keys {
				if m > 0 {
					n = fmt.Sprintf("[%s]", n)
				}

				tmp = append(tmp, n)
			}

			kStr := strings.Join(tmp, "")

			p[fmt.Sprintf("%s[%s]", kStr, k)] = fmt.Sprintf("%s", vv)

			break
		}
	}
}

func parseNormal(p map[string]string, raw []interface{}, keys []string) {

	for k, v := range raw {
		switch vv := v.(type) {
		case []interface{}:

			tmpKeys := append(keys, fmt.Sprintf("%d", k))

			parseNormal(p, vv, tmpKeys)

			break
		case map[string]interface{}:

			tmpKeys := append(keys, fmt.Sprintf("%d", k))

			parseKeyValue(p, vv, tmpKeys)

			break
		default:

			//keys = append(keys, fmt.Sprintf("%d", k))

			var tmp []string

			for m, n := range keys {
				if m > 0 {
					n = fmt.Sprintf("[%s]", n)
				}

				tmp = append(tmp, n)
			}

			kStr := strings.Join(tmp, "")

			p[fmt.Sprintf("%s[%d]", kStr, k)] = fmt.Sprintf("%s", vv)

			break
		}
	}
}
