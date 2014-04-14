// Copyright 2014, Orchestrate.IO, Inc.

package client

import (
	"encoding/json"
	"strings"
)

// Holds results returned from a Graph query.
type GraphResults struct {
	Count   uint64        `json:"count"`
	Results []GraphResult `json:"results"`
}

// An individual graph result.
type GraphResult struct {
	Path     Path            `json:"path"`
	RawValue json.RawMessage `json:"value"`
}

// Get all related key/value objects by collection-key and a list of relations.
func (c *Client) GetRelations(collection string, key string, hops []string) (*GraphResults, error) {
	relationsPath := strings.Join(hops, "/")

	trailingUri := collection + "/" + key + "/relations/" + relationsPath
	resp, err := c.doRequest("GET", trailingUri, nil, nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, newError(resp)
	}

	decoder := json.NewDecoder(resp.Body)
	result := new(GraphResults)
	if err := decoder.Decode(result); err != nil {
		return nil, err
	}

	return result, nil
}

// Create a relationship of a specified type between two collection-keys.
func (c *Client) PutRelation(sourceCollection string, sourceKey string, kind string, sinkCollection string, sinkKey string) error {
	trailingUri := sourceCollection + "/" + sourceKey + "/relation/" + kind + "/" + sinkCollection + "/" + sinkKey
	resp, err := c.doRequest("PUT", trailingUri, nil, nil)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return newError(resp)
	}
	return nil
}

// Marshall the value of a GraphResult into the provided object.
func (result *GraphResult) Value(value interface{}) error {
	return json.Unmarshal(result.RawValue, value)
}
