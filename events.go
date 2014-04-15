// Copyright 2014, Orchestrate.IO, Inc.

package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"strconv"
)

// Holds results returned from an Events query.
type EventResults struct {
	Count   uint64  `json:"count"`
	Results []Event `json:"results"`
}

// An individual event.
type Event struct {
	Ordinal   uint64          `json:"ordinal"`
	Timestamp uint64          `json:"timestamp"`
	RawValue  json.RawMessage `json:"value"`
}

// Get latest events of a particular type from specified collection-key pair.
func (c *Client) GetEvents(collection string, key string, kind string) (*EventResults, error) {
	trailingUri := collection + "/" + key + "/events/" + kind

	return c.doGetEvents(trailingUri)
}

// Get all events of a particular type from specified collection-key pair in a
// range.
func (c *Client) GetEventsInRange(collection string, key string, kind string, start int64, end int64) (*EventResults, error) {
	queryVariables := url.Values{
		"start": []string{strconv.FormatInt(start, 10)},
		"end":   []string{strconv.FormatInt(end, 10)},
	}

	trailingUri := collection + "/" + key + "/events/" + kind + "?" + queryVariables.Encode()

	return c.doGetEvents(trailingUri)
}

// Put an event of the specified type to provided collection-key pair.
func (c *Client) PutEvent(collection, key, kind string, value interface{}) error {
	buf := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(value); err != nil {
		return err
	}

	return c.PutEventRaw(collection, key, kind, buf)
}

// Put an event of the specified type to provided collection-key pair.
func (c *Client) PutEventRaw(collection, key, kind string, value io.Reader) error {
	trailingUri := collection + "/" + key + "/events/" + kind

	return c.doPutEvent(trailingUri, value)

}

// Put an event of the specified type to provided collection-key pair and time.
func (c *Client) PutEventWithTime(collection, key, kind string, time int64, value interface{}) error {
	buf := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buf)

	if err := encoder.Encode(value); err != nil {
		return err
	}

	return c.PutEventWithTimeRaw(collection, key, kind, time, buf)
}

// Put an event of the specified type to provided collection-key pair and time.
func (c *Client) PutEventWithTimeRaw(collection, key, kind string, time int64, value io.Reader) error {
	queryVariables := url.Values{
		"timestamp": []string{strconv.FormatInt(time, 10)},
	}

	trailingUri := collection + "/" + key + "/events/" + kind + "?" + queryVariables.Encode()

	return c.doPutEvent(trailingUri, value)
}

// Execute event get.
func (c *Client) doGetEvents(trailingUri string) (*EventResults, error) {
	resp, err := c.doRequest("GET", trailingUri, nil, nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, newError(resp)
	}

	decoder := json.NewDecoder(resp.Body)
	results := new(EventResults)
	if err = decoder.Decode(results); err != nil {
		return nil, err
	}

	return results, err
}

// Execute event put.
func (c *Client) doPutEvent(trailingUri string, value io.Reader) error {
	resp, err := c.doRequest("PUT", trailingUri, nil, value)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return newError(resp)
	}
	return nil
}

// Marshall the value of an event into the provided object.
func (r *Event) Value(value interface{}) error {
	return json.Unmarshal(r.RawValue, value)
}
