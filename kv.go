// Copyright 2014, Orchestrate.IO, Inc.

package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"strconv"
	"strings"
)

// Holds results returned from a KV list query.
type KVResults struct {
	Count   uint64     `json:"count"`
	Results []KVResult `json:"results"`
	Next    string     `json:"next,omitempty"`
}

// An individual Key/Value result.
type KVResult struct {
	Path     Path            `json:"path"`
	RawValue json.RawMessage `json:"value"`
}

// Get a collection-key pair's value.
func (client *Client) Get(collection, key string) (*KVResult, error) {
	return client.GetPath(&Path{Collection: collection, Key: key})
}

// Get the value at a path.
func (client *Client) GetPath(path *Path) (*KVResult, error) {
	resp, err := client.doRequest("GET", path.trailingGetURI(), nil, nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, newError(resp)
	}

	// TODO: Check for a content-length header so we can pre-allocate buffer
	// space.
	buf := bytes.NewBuffer(nil)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, err
	}

	if path.Ref == "" {
		path.Ref = strings.SplitAfter(resp.Header.Get("Content-Location"), "/")[5]
	}

	return &KVResult{Path: *path, RawValue: buf.Bytes()}, nil
}

// Store a value to a collection-key pair.
func (client *Client) Put(collection string, key string, value interface{}) (*Path, error) {
	buf := new(bytes.Buffer)
	encoder := json.NewEncoder(buf)

	if err := encoder.Encode(value); err != nil {
		return nil, err
	}

	return client.PutRaw(collection, key, buf)
}

// Store a value to a collection-key pair.
func (client *Client) PutRaw(collection string, key string, value io.Reader) (*Path, error) {
	return client.doPut(&Path{Collection: collection, Key: key}, nil, value)
}

// Store a value to a collection-key pair if the path's ref value is the latest.
func (client *Client) PutIfUnmodified(path *Path, value interface{}) (*Path, error) {
	buf := new(bytes.Buffer)
	encoder := json.NewEncoder(buf)

	if err := encoder.Encode(value); err != nil {
		return nil, err
	}

	return client.PutIfUnmodifiedRaw(path, buf)
}

// Store a value to a collection-key pair if the path's ref value is the latest.
func (client *Client) PutIfUnmodifiedRaw(path *Path, value io.Reader) (*Path, error) {
	headers := map[string]string{
		"If-Match": "\""+path.Ref+"\"",
	}

	return client.doPut(path, headers, value)
}

// Store a value to a collection-key pair if it doesn't already hold a value.
func (client *Client) PutIfAbsent(collection string, key string, value interface{}) (*Path, error) {
	buf := new(bytes.Buffer)
	encoder := json.NewEncoder(buf)

	if err := encoder.Encode(value); err != nil {
		return nil, err
	}

	return client.PutIfAbsentRaw(collection, key, buf)
}

// Store a value to a collection-key pair if it doesn't already hold a value.
func (client *Client) PutIfAbsentRaw(collection string, key string, value io.Reader) (*Path, error) {
	headers := map[string]string{
		"If-None-Match": "\"*\"",
	}

	return client.doPut(&Path{Collection: collection, Key: key}, headers, value)
}

func (client *Client) doPut(path *Path, headers map[string]string, value io.Reader) (*Path, error) {
	resp, err := client.doRequest("PUT", path.trailingPutURI(), headers, value)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return nil, newError(resp)
	}

	ref := strings.SplitAfter(resp.Header.Get("Location"), "/")[5]

	return &Path{
		Collection: path.Collection,
		Key:        path.Key,
		Ref:        ref,
	}, err
}

// Delete the value held at a collection-key pair.
func (client *Client) Delete(collection, key string) error {
	return client.doDelete(collection+"/"+key, nil)
}

// Delete the value held at a collection-key par if the path's ref value is the
// latest.
func (client *Client) DeleteIfUnmodified(path *Path) error {
	headers := map[string]string{
		"If-Match": "\""+path.Ref+"\"",
	}

	return client.doDelete(path.trailingPutURI(), headers)
}

// Delete the current and all previous values from a collection-key pair.
func (client *Client) Purge(collection, key string) error {
	return client.doDelete(collection+"/"+key+"?purge=true", nil)
}

// Delete a collection.
func (client *Client) DeleteCollection(collection string) error {
	return client.doDelete(collection+"?force=true", nil)
}

// Execute delete
func (client *Client) doDelete(trailingUri string, headers map[string]string) error {
	resp, err := client.doRequest("DELETE", trailingUri, headers, nil)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return newError(resp)
	}

	return nil
}

// List the values in a collection in key order with the specified page size.
func (client *Client) List(collection string, limit int) (*KVResults, error) {
	queryVariables := url.Values{
		"limit": []string{strconv.Itoa(limit)},
	}

	trailingUri := collection+"?"+queryVariables.Encode()

	return client.doList(trailingUri)
}

// List the values in a collection in key order with the specified page size
// that come after the specified key.
func (client *Client) ListAfter(collection string, after string, limit int) (*KVResults, error) {
	queryVariables := url.Values{
		"limit":    []string{strconv.Itoa(limit)},
		"afterKey": []string{after},
	}

	trailingUri := collection+"?"+queryVariables.Encode()

	return client.doList(trailingUri)
}

// List the values in a collection in key order with the specified page size
// starting with the specified key.
func (client *Client) ListStart(collection string, start string, limit int) (*KVResults, error) {
	queryVariables := url.Values{
		"limit":    []string{strconv.Itoa(limit)},
		"startKey": []string{start},
	}

	trailingUri := collection+"?"+queryVariables.Encode()

	return client.doList(trailingUri)
}

// Get the page of key/value list results that follow that provided set.
func (client *Client) ListGetNext(results *KVResults) (*KVResults, error) {
	return client.doList(results.Next[4:])
}

// Execute a key/value list operation.
func (client *Client) doList(trailingUri string) (*KVResults, error) {
	resp, err := client.doRequest("GET", trailingUri, nil, nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, newError(resp)
	}

	decoder := json.NewDecoder(resp.Body)
	result := new(KVResults)
	if err := decoder.Decode(result); err != nil {
		return result, err
	}

	return result, nil
}

// Check if there is a subsequent page of key/value list results.
func (results *KVResults) HasNext() bool {
	return results.Next != ""
}

// Marshall the value of a KVResult into the provided object.
func (result *KVResult) Value(value interface{}) error {
	return json.Unmarshal(result.RawValue, value)
}

// Returns the trailing URI part for a GET request.
func (path *Path) trailingGetURI() string {
	if path.Ref != "" {
		return path.Collection+"/"+path.Key+"/refs/"+path.Ref
	}
	return path.Collection+"/"+path.Key
}

// Returns the trailing URI part for a PUT request.
func (path *Path) trailingPutURI() string {
	return path.Collection+"/"+path.Key
}
