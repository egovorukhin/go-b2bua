package sippy_header

import (
	"strings"
)

type tagListBody struct {
	Tags []string
}

func newTagListBody(body string) *tagListBody {
	tags := make([]string, 0)
	for _, s := range strings.Split(body, ",") {
		tags = append(tags, strings.TrimSpace(s))
	}
	return &tagListBody{
		Tags: tags,
	}
}

func (s *tagListBody) String() string {
	return strings.Join(s.Tags, ", ")
}

func (s *tagListBody) HasTag(tag string) bool {
	for _, storedTag := range s.Tags {
		if storedTag == tag {
			return true
		}
	}
	return false
}

type tagListHF struct {
	stringBody string
	body       *tagListBody
}

func createTagListHF(body string) *tagListHF {
	return &tagListHF{
		stringBody: body,
	}
}

func (s *tagListHF) GetBody() *tagListBody {
	if s.body == nil {
		s.body = newTagListBody(s.stringBody)
	}
	return s.body
}

func (s *tagListHF) getCopy() *tagListHF {
	tmp := *s
	if s.body != nil {
		body := *s.body
		tmp.body = &body
	}
	return &tmp
}

func (s *tagListHF) StringBody() string {
	if s.body != nil {
		return s.body.String()
	}
	return s.stringBody
}

func (s *tagListHF) HasTag(tag string) bool {
	body := s.GetBody()
	return body.HasTag(tag)
}
