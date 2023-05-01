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
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// PinotHTTP interface
type PinotHTTP interface {
	Do() *Response
}

// HTTP client
type Client struct {
	Method     string
	URL        string
	HTTPClient http.Client
	Body       []byte
	Auth       Auth
}

func NewHTTPClient(method, url string, client http.Client, body []byte, auth Auth) PinotHTTP {
	newClient := &Client{
		Method:     method,
		URL:        url,
		HTTPClient: client,
		Body:       body,
		Auth:       auth,
	}

	return newClient
}

// Auth mechanisms supported by pinot control plane to authenticate
// with pinot clusters
type Auth struct {
	BasicAuth BasicAuth
}

// BasicAuth
type BasicAuth struct {
	UserName string
	Password string
}

// Pinot API error Response
// ex: {"code":404,"error":"Schema not found"}
type PinotErrorResponse struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

// Pinot API success Response
// ex: {"unrecognizedProperties":{},"status":"airlineStats successfully added"}
type PinotSuccessResponse struct {
	UnrecognizedProperties interface{} `json:"unrecognizedProperties"`
	Status                 string      `json:"status"`
}

// Response passed to controller
type Response struct {
	Err        error
	StatusCode int
	PinotErrorResponse
	PinotSuccessResponse
}

// Initiate HTTP call to pinot
func (c *Client) Do() *Response {

	req, err := http.NewRequest(c.Method, c.URL, bytes.NewBuffer(c.Body))
	if err != nil {
		return &Response{Err: err}
	}

	if c.Auth.BasicAuth != (BasicAuth{}) {
		req.SetBasicAuth(c.Auth.BasicAuth.UserName, c.Auth.BasicAuth.Password)
	}

	req.Header.Add("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return &Response{Err: err}
	}

	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &Response{Err: err}
	}

	// GET /schemas returns 404 when schema not found with code and error as resp.
	// GET /tenants returns 404 when tenant not found with code and error as resp
	// GET /tables returns 200 when table not found with an empty response.
	if string(responseBody) != "{}" {
		if resp.StatusCode == 200 {
			var pinotSuccess PinotSuccessResponse
			if err := json.Unmarshal(responseBody, &pinotSuccess); err != nil {
				return &Response{Err: err}
			}
			return &Response{StatusCode: resp.StatusCode, PinotSuccessResponse: pinotSuccess}
		} else {
			var pinotErr PinotErrorResponse
			if err := json.Unmarshal(responseBody, &pinotErr); err != nil {
				return &Response{StatusCode: resp.StatusCode, Err: err}
			}
			return &Response{StatusCode: resp.StatusCode, PinotErrorResponse: pinotErr}
		}
	} else {
		if resp.StatusCode == 200 {
			// resp is empty with 200 status code
			// for tables API force 404
			return &Response{StatusCode: 404}
		} else {
			var pinotErr PinotErrorResponse
			if err := json.Unmarshal(responseBody, &pinotErr); err != nil {
				return &Response{StatusCode: resp.StatusCode, Err: err}
			}
			return &Response{StatusCode: resp.StatusCode, PinotErrorResponse: pinotErr}
		}
	}
}
