package indextank

import (
	"appengine"
	"appengine/urlfetch"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const version = "0.3"
const APP_ENGINE_USED = true
const userAgent = "Searchify-Gotank/" + version

func makeIndexUrl(apiUrl, name string) string {
	return fmt.Sprintf("%s/v1/indexes/%s", apiUrl, name)
}

func request(method, uri string, data interface{}) (*http.Response, error) {
	method = strings.ToUpper(method)

	var bodyReader io.Reader = nil
	var contentLength int64 = 0
	var httpClient *http.Client
	if data != nil {
		b, err := json.Marshal(data)
		contentLength = int64(len(b))
		if err != nil {
			//fmt.Println("Error marshalling: %v\n", err)
			return nil, err
		}
		//fmt.Println("  Marshalled request: ", string(b))
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, uri, bodyReader)
	if err != nil {
		return nil, err
	}

	if APP_ENGINE_USED {
		context := appengine.NewContext(req)
		httpClient = urlfetch.Client(context)
	} else {
		httpClient = http.DefaultClient
	}

	if method == "POST" || method == "PUT" || (method == "DELETE" && contentLength > 0) {
		//fmt.Printf("Setting content-length to %d for %s %s\n", contentLength, method, uri)
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = contentLength
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	// make sure the caller calls resp.Body.Close() if necessary
	return resp, err
}

func doRequest(method, requestUrl string, params map[string]string) (map[string]interface{}, error) {
	// caller must construct url
	uri := requestUrl

	queryString := toQueryString(params)
	uri += "?" + queryString
	//fmt.Printf("---------> %s\n", queryString)

	resp, err := request(method, uri, nil)

	defer resp.Body.Close()
	//fmt.Printf(" [status %d]\n", resp.StatusCode)
	if resp.StatusCode == 404 {
		return nil, errors.New("Index does not exist")
	}
	if resp.StatusCode == 204 {
		return nil, errors.New("Index Already Exists " + strconv.Itoa(resp.StatusCode))
	}
	if resp.StatusCode >= 400 {
		return nil, errors.New("HTTP response " + strconv.Itoa(resp.StatusCode))
	}
	body, err := ioutil.ReadAll(resp.Body)
	//fmt.Printf("* ReadAll err: %v, body length = %d\n", err, len(body))
	//fmt.Printf("  BODY: %v\n", string(body))
	if body == nil || len(body) == 0 {
		return nil, err
	}

	var i interface{}
	err = json.Unmarshal(body, &i)

	m := i.(map[string]interface{})
	return m, err
}

func toQueryString(params map[string]string) string {
	s := ""
	if params == nil {
		return s
	}
	for k, v := range params {
		s += k + "=" + url.QueryEscape(v) + "&"
	}
	s = s[0 : len(s)-1]
	return s
}

func readResponseBody(resp *http.Response) (string, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func isOk(status int) bool {
	return status/100 == 2
}
