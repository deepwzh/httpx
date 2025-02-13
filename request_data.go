package httpx

import (
	"fmt"
	"net/url"

	"github.com/bytedance/sonic"
)

type RequestData interface {
	Marshal() ([]byte, error)
}

var _ RequestData = (*JsonRequestData)(nil)

type JsonRequestData struct {
	data any
}

func (j *JsonRequestData) Marshal() ([]byte, error) {
	buf, err := sonic.Marshal(j.data)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

type QueryRequestData struct {
	data url.Values
}

type RawRequestData struct {
	data []byte
}

func (r *RawRequestData) Marshal() ([]byte, error) {
	return r.data, nil
}

func NewRawData(data []byte) RequestData {
	return &RawRequestData{data: data}
}

func NewQueryData(data url.Values) RequestData {
	return &QueryRequestData{data: data}
}

func (q *QueryRequestData) Marshal() ([]byte, error) {
	postData := url.Values{}
	for k, v := range q.data {
		postData.Add(k, fmt.Sprintf("%v", v))
	}
	return []byte(postData.Encode()), nil
}

func NewJsonData(data any) RequestData {
	return &JsonRequestData{data: data}
}

// func New(data map[string]any) RequestData {
// 	return &JsonRequestData{data: data}
// }
