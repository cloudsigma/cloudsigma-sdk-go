package cloudsigma

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

const tagsBasePath = "tags"

// TagsService handles communication with the tags related methods of the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/tags.html
type TagsService service

type Tag struct {
	Meta        TagMeta `json:"meta,omitempty"`
	Name        string  `json:"name,omitempty"`
	Owner       Owner   `json:"owner,omitempty"`
	ResourceURI string  `json:"resource_uri,omitempty"`
	UUID        string  `json:"uuid,omitempty"`
}

type TagMeta struct {
	Color string `json:"color,omitempty"`
}

type TagCreateRequest struct {
	Tags []Tag `json:"objects"`
}

type tagsRoot struct {
	Tags []Tag `json:"objects"`
}

func (s *TagsService) List(ctx context.Context) ([]Tag, *http.Response, error) {
	path := fmt.Sprintf("%v", tagsBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(tagsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Tags, resp, err
}

func (s *TagsService) Get(ctx context.Context, uuid string) (*Tag, *http.Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v", tagsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	tag := new(Tag)
	resp, err := s.client.Do(ctx, req, tag)
	if err != nil {
		return nil, resp, err
	}

	return tag, resp, err
}

func (s *TagsService) Create(ctx context.Context, tagCreateRequest *TagCreateRequest) (*Tag, *http.Response, error) {
	if tagCreateRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/", tagsBasePath)

	req, err := s.client.NewRequest(http.MethodPost, path, tagCreateRequest)
	if err != nil {
		return nil, nil, err
	}

	root := new(TagCreateRequest)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	if len(root.Tags) > 1 {
		return nil, resp, errors.New("root.Tags count cannot be more than 1")
	}

	return &root.Tags[0], resp, err
}

func (s *TagsService) Update(ctx context.Context, tag *Tag) (*Tag, *http.Response, error) {
	if tag == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/%v/", tagsBasePath, tag.UUID)

	req, err := s.client.NewRequest(http.MethodPut, path, tag)
	if err != nil {
		return nil, nil, err
	}

	root := new(Tag)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, err
}

func (s *TagsService) Delete(ctx context.Context, uuid string) (*http.Response, error) {
	if uuid == "" {
		return nil, ErrEmptyArgument
	}

	path := fmt.Sprintf("%v/%v/", tagsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
