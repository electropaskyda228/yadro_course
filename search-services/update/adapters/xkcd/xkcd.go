package xkcd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"yadro.com/course/update/core"
)

const urlEnd = "/info.0.json"

type Client struct {
	log    *slog.Logger
	client http.Client
	url    string
}

func NewClient(url string, timeout time.Duration, log *slog.Logger) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("empty base url specified")
	}
	return &Client{
		client: http.Client{Timeout: timeout},
		log:    log,
		url:    url,
	}, nil
}

func (c Client) Get(ctx context.Context, id int) (core.XKCDInfo, error) {
	url := c.url + "/" + strconv.Itoa(id) + urlEnd

	c.log.Info("Building request to xkcd on url: " + url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.log.Error("failed to build request to xkcd", "error", err)
		return core.XKCDInfo{}, err
	}

	c.log.Info("Send request to url: " + url)
	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Error("Request to url: "+url+" failed", "error", err)
		return core.XKCDInfo{}, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("Failed to close request connection", "error", err)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return core.XKCDInfo{}, core.ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		c.log.Error("Got strange status code in request for getting comics id: "+strconv.Itoa(id), "error", err)
		return core.XKCDInfo{}, errors.New("unknown status: " + strconv.Itoa(resp.StatusCode))
	}

	var rawData map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		c.log.Error("Failed to parse json of comics id: "+strconv.Itoa(id), "error", err)
		return core.XKCDInfo{}, err
	}

	info := core.XKCDInfo{
		ID:          int(rawData["num"].(float64)),
		URL:         rawData["img"].(string),
		Title:       rawData["title"].(string),
		Description: rawData["alt"].(string),
		SafeTitle:   rawData["safe_title"].(string),
		Transcript:  rawData["transcript"].(string),
	}

	c.log.Info("All information about comics id: " + strconv.Itoa(id) + "have been gotten")

	return info, nil
}

func (c Client) LastID(ctx context.Context) (int, error) {
	url := c.url + urlEnd

	c.log.Info("Send request in purpose of getting last comics id")
	resp, err := c.client.Get(url)
	if err != nil {
		c.log.Error("Failed to send request for getting last comics id", "error", err)
		return -1, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("Failed to close request connection", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		c.log.Error("Got strange status code in request for getting last comics id", "error", err)
		return -1, err
	}

	var rawData map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
		c.log.Error("Failed to parse json of last id comics", "error", err)
		return -1, err
	}

	id := int(rawData["num"].(float64))

	c.log.Info("Last id of comics' has been gotten: " + strconv.Itoa(id))

	return id, nil
}
