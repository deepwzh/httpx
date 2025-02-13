package httpx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	nurl "net/url"
	"time"

	"github.com/bytedance/sonic"
)

type Responser interface {
}
type Response struct {
	response *http.Response
	rawdata  []byte
	close    bool
}

func (resp *Response) GetRawResponse() *http.Response {
	return resp.response
}

func NewResponse(response *http.Response) *Response {
	if response == nil {
		return nil
	}
	return &Response{response: response, close: false}
}

func (resp *Response) Close() error {
	return resp.response.Body.Close()
}

// type Jsonify[T any] interface {
// 	Json(data T) (*T, error)
// }

func (resp *Response) Json(res any) error {
	if resp.close {
		if err := sonic.Unmarshal(resp.rawdata, &res); err != nil {
			return err
		}
		return nil
	} else {
		body, err := io.ReadAll(resp.response.Body)
		if err != nil {
			return err
		}

		resp.close = true
		resp.rawdata = body

		if err := sonic.Unmarshal(body, &res); err != nil {
			return err
		}
		return nil
	}
}

func (resp *Response) Text() []byte {
	if resp.close {
		return resp.rawdata
	} else {
		body, err := io.ReadAll(resp.response.Body)
		if err != nil {
			slog.Error("read body failed", "err",
				err)
			return nil
		}

		resp.close = true
		resp.rawdata = body

		return body
	}
}

func (resp *Response) Status() int {
	return resp.response.StatusCode
}

// Name server name
func RequestTimeoutOption(param time.Duration) ClientOption {
	return func(o *Client) {
		o.Timeout = param
	}
}

func WithRequest(client *http.Client) ClientOption {
	return func(o *Client) {
		o.client = client
	}
}

const (
	ContentTypeJson       = "application/json"
	ContentTypeForm       = "application/x-www-form-urlencoded"
	ContentTypeHeaderName = "Content-Type"
)

func WithContentType(contentType string) ClientOption {
	return func(o *Client) {
		o.Header[ContentTypeHeaderName] = contentType
	}
}

func WithHeader(header map[string]string) ClientOption {
	return func(o *Client) {
		o.Header = header
	}
}

func WithRetryConfig(c *RetryConfig) ClientOption {
	return func(o *Client) {
		o.retryConfig = c
	}
}

//	func PreRequestCallback(*http.Request) error {
//		return nil
//	}
func PreRequestCallbackOption(callback func(*http.Request, string) error) ClientOption {
	return func(o *Client) {
		o.PreRequestCallback = callback
	}
}

func WithCookies(cookies []*http.Cookie) ClientOption {
	return func(o *Client) {
		o.cookies = cookies
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(o *Client) {
		o.Timeout = timeout
	}
}

type ClientOption func(*Client)

func (client *Client) DoRequest(url string, data RequestData, method string, opts ...RequestOption) (*Response, error) {
	opt := &RequestOptions{}
	for _, o := range opts {
		o(opt)
	}

	header := make(http.Header)
	for k, v := range opt.Header {
		header[k] = []string{v}
	}

	for k, v := range client.Header {
		header[k] = []string{v}
	}

	var body io.Reader
	var bodyStr string

	if data != nil {
		buf, err := data.Marshal()
		if err != nil {
			return nil, fmt.Errorf("marshal data failed: %v", err)
		}
		body = bytes.NewBuffer(buf)
		bodyStr = string(buf)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	for _, cookie := range client.cookies {
		req.AddCookie(cookie)
	}

	if client.PreRequestCallback != nil {
		client.PreRequestCallback(req, bodyStr)
	}

	req.Header = header

	slog.Debug("http request", "method", method, "url", url, "body", bodyStr, "header", req.Header)

	resp, err := doRequestWithRetry(client.client, req, client.retryConfig)
	if err != nil {
		return nil, err
	}
	return NewResponse(resp), nil
}

func MustParseStructToMap(data interface{}) map[string]interface{} {
	res, err := struct2Map(data)
	if err != nil {
		panic(err)
	}
	return res
}

//	type Client interface {
//		QueryParamToString(query map[string]interface{}) (v nurl.Values)
//		Get(url string, query nurl.Values) (*Response, error)
//		Post(url string, data map[string]interface{}) (*Response, error)
//		Put(url string, data map[string]interface{}) (*Response, error)
//		Patch(url string, data map[string]interface{}) (*Response, error)
//		Delete(url string, data map[string]interface{}) (*Response, error)
//		Head(url string) (*Response, error)
//	}
type Request struct {
	Header map[string]string
}

type Client struct {
	client             *http.Client
	PreRequestCallback func(*http.Request, string) error
	Timeout            time.Duration
	Header             map[string]string
	cookies            []*http.Cookie
	retryConfig        *RetryConfig
}

func NewClient(opts ...ClientOption) *Client {
	client := &Client{
		Timeout: 10 * time.Second,
	}
	for _, o := range opts {
		o(client)
	}

	if client.client == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			slog.Error("create cookie jar failed", "err", err)
		}
		client.client = &http.Client{
			Jar: jar,
		}
	}

	if client.Timeout != 0 {
		client.client.Timeout = client.Timeout
	}

	return client
}

// func (req *Request) SetHeader(name string, value string) error {
// 	req.Header[name] = value
// 	return nil
// }

var (
	DefaultRequest = &Request{}
	// JsonRequest    = RequestWithHeader(&map[string]string{
	// 	"Content-Type": "application/json",
	// })

	// FormRequest = RequestWithHeader(&map[string]string{
	// 	"Content-Type": "application/x-www-form-urlencoded",
	// })
)

func IsTimeout(err error) bool {
	if neterr := (net.Error)(nil); errors.As(err, &neterr) {
		return neterr.Timeout()
	}
	return false
}

func NewClientWithHeader(header map[string]string) *Client {
	return NewClient(WithHeader(header))
}

func (req *Client) QueryParamToString(query map[string]interface{}) (v nurl.Values) {
	for key, item := range query {
		v.Add(key, fmt.Sprintf("%v", item))
	}
	return
}

type RequestOptions struct {
	Header map[string]string
}

type RequestOption func(*RequestOptions)

func RequestHeader(header map[string]string) RequestOption {
	return func(o *RequestOptions) {
		o.Header = header
	}
}

func (c *Client) Get(url string, query RequestParam, opts ...RequestOption) (*Response, error) {
	if query != nil {
		urlWithQuery := fmt.Sprintf("%s?%s", url, query.Marshal())
		return c.DoRequest(urlWithQuery, nil, http.MethodGet, opts...)
	} else {
		return c.DoRequest(url, nil, http.MethodGet, opts...)
	}
}

func (c *Client) Post(url string, data RequestData, opts ...RequestOption) (*Response, error) {
	return c.DoRequest(url, data, http.MethodPost, opts...)
}

func (c *Client) Put(url string, data RequestData, opts ...RequestOption) (*Response, error) {
	return c.DoRequest(url, data, http.MethodPut, opts...)
}

func (c *Client) Patch(url string, data RequestData, opts ...RequestOption) (*Response, error) {
	return c.DoRequest(url, data, http.MethodPatch, opts...)
}

func (c *Client) Delete(url string, data RequestData, opts ...RequestOption) (*Response, error) {
	return c.DoRequest(url, data, http.MethodDelete, opts...)
}

func (c *Client) Head(url string, opts ...RequestOption) (*Response, error) {
	return c.DoRequest(url, nil, http.MethodHead, opts...)
}
