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

// PinotHTTP interface
type PinotHTTP interface {
	Do() (*Response, error)
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

// Response passed to controller
type Response struct {
	ResponseBody string
	StatusCode   int
}

// GET /schemas returns 404 when schema not found with code and error as resp.
// GET /tenants returns 404 when tenant not found with code and error as resp
// GET /tables returns 200 when table not found with an empty response.

// Do method to be used schema and tenant controller.
func (c *Client) Do() (*Response, error) {

	req, err := http.NewRequest(c.Method, c.URL, bytes.NewBuffer(c.Body))
	if err != nil {
		return nil, err
	}

	if c.Auth.BasicAuth != (BasicAuth{}) {
		req.SetBasicAuth(c.Auth.BasicAuth.UserName, c.Auth.BasicAuth.Password)
	}

	req.Header.Add("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &Response{ResponseBody: string(responseBody), StatusCode: resp.StatusCode}, nil
}
