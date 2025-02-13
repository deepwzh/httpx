package httpx

import (
	"net/url"
)

type RequestParam interface {
	Marshal() string
}

var _ RequestData = (*JsonRequestData)(nil)

func NewUrlRequestParam(data url.Values) RequestParam {
	return &UrlRequestParam{data: data}
}

type UrlRequestParam struct {
	data url.Values
}

func (j *UrlRequestParam) Marshal() string {
	buf := j.data.Encode()
	return buf
}

func NewMapParam(data map[string]any) RequestParam {
	return &MapRequestParam{data: data}
}

type MapRequestParam struct {
	data map[string]any
}

func (j *MapRequestParam) Marshal() string {
	d := ""
	for k, v := range j.data {
		d += k + "=" + v.(string) + "&"
	}
	return d
}

type RawRequestParam struct {
	data string
}

func (r *RawRequestParam) Marshal() string {
	return r.data
}

func NewRawParam(data string) RequestParam {
	return &RawRequestParam{data: data}
}
