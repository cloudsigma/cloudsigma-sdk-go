package cloudsigma

import (
	"context"
	"fmt"
	"net/http"
)

const tagsBasePath = "tags"

// TagsService handles communication with the tags related methods of the CloudSigma API.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/tags.html
type TagsService service

// Tag represents a CloudSigma tag.
type Tag struct {
	Meta        map[string]interface{} `json:"meta,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Owner       ResourceLink           `json:"owner,omitempty"`
	ResourceURI string                 `json:"resource_uri,omitempty"`
	Resources   []TagResource          `json:"resources,omitempty"`
	UUID        string                 `json:"uuid"`
}

// TagResource represents a resource assigned to the tag.
type TagResource struct {
	Owner        ResourceLink `json:"owner,omitempty"`
	ResourceType string       `json:"res_type,omitempty"`
	ResourceURI  string       `json:"resource_uri,omitempty"`
	UUID         string       `json:"uuid"`
}

// TagCreateRequest represents a request to create a tag.
type TagCreateRequest struct {
	Tags []Tag `json:"objects"`
}

// TagUpdateRequest represents a request to update a tag.
type TagUpdateRequest struct {
	*Tag
}

type tagsRoot struct {
	Meta *Meta `json:"meta,omitempty"`
	Tags []Tag `json:"objects"`
}

// List provides a list of tags to which the authenticated user has access.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/tags.html#listing
func (s *TagsService) List(ctx context.Context) ([]Tag, *Response, error) {
	path := fmt.Sprintf("%v/", tagsBasePath)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(tagsRoot)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}
	if m := root.Meta; m != nil {
		resp.Meta = m
	}

	return root.Tags, resp, nil
}

// Get provides detailed information for tag identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/tags.html#list-single-tag
func (s *TagsService) Get(ctx context.Context, uuid string) (*Tag, *Response, error) {
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

	return tag, resp, nil
}

// Create makes a new tag (or tags) with given payload.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/tags.html#creating
func (s *TagsService) Create(ctx context.Context, tagCreateRequest *TagCreateRequest) ([]Tag, *Response, error) {
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

	return root.Tags, resp, nil
}

// Update edits a tag identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/tags.html#editing
func (s *TagsService) Update(ctx context.Context, uuid string, tagUpdateRequest *TagUpdateRequest) (*Tag, *Response, error) {
	if uuid == "" {
		return nil, nil, ErrEmptyArgument
	}
	if tagUpdateRequest == nil {
		return nil, nil, ErrEmptyPayloadNotAllowed
	}

	path := fmt.Sprintf("%v/%v/", tagsBasePath, uuid)

	req, err := s.client.NewRequest(http.MethodPut, path, tagUpdateRequest)
	if err != nil {
		return nil, nil, err
	}

	tag := new(Tag)
	resp, err := s.client.Do(ctx, req, tag)
	if err != nil {
		return nil, resp, err
	}

	return tag, resp, nil
}

// Delete removes a single tag identified by uuid.
//
// CloudSigma API docs: https://cloudsigma-docs.readthedocs.io/en/latest/tags.html#deleting
func (s *TagsService) Delete(ctx context.Context, uuid string) (*Response, error) {
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
