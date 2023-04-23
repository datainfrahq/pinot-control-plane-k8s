/*
DataInfra Pinot Control Plane (C) 2023 - 2024 DataInfra.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package http

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

type PinotHTTP interface {
	Do() HttpResponse
}

type Client struct {
	Method     string
	URL        string
	HTTPClient http.Client
	Body       []byte
}

type HttpResponse struct {
	RespBody   []byte
	Err        error
	StatusCode int
}

func NewHTTPClient(method, url string, client http.Client, body []byte) PinotHTTP {
	newClient := &Client{
		Method:     method,
		URL:        url,
		HTTPClient: client,
		Body:       body,
	}

	return newClient
}

func (c *Client) Do() HttpResponse {

	req, err := http.NewRequest(c.Method, c.URL, bytes.NewBuffer(c.Body))
	if err != nil {
		return HttpResponse{Err: err}
	}

	req.Header.Add("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return HttpResponse{Err: err}
	}

	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return HttpResponse{Err: err}
	}

	return HttpResponse{RespBody: responseBody, Err: nil, StatusCode: resp.StatusCode}

}
