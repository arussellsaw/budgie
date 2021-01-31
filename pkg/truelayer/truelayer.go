package truelayer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"golang.org/x/oauth2"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/token"
)

const (
	// sandbox credentials
	// TODO: not this
	clientID     = "sandbox-sheets-35b0b7"
	clientSecret = "2b1dba10-b1aa-434e-9dd0-b0ee11e84293"

	baseURL = "https://api.truelayer.com"
)

func NewClient(ctx context.Context, userID string) (*Client, error) {
	t, err := token.Get(ctx, OauthConfig, userID)
	if err != nil {
		return nil, err
	}
	return &Client{
		userID: userID,
		t:      t,
		http: &http.Client{
			Transport: http.DefaultTransport,
			Timeout:   10 * time.Second,
		},
	}, nil
}

type Client struct {
	userID string
	t      *oauth2.Token
	http   *http.Client
}

func (c *Client) authRequest(r *http.Request) {
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.t.AccessToken))
}

func (c *Client) Accounts(ctx context.Context) ([]Account, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/data/v1/accounts", baseURL),
		nil,
	)
	if err != nil {
		return nil, err
	}
	c.authRequest(req)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	response := struct {
		Results []Account `json:"results"`
		Error   string
	}{}
	err = json.NewDecoder(res.Body).Decode(&response)
	return response.Results, err
}

func (c *Client) Transactions(ctx context.Context, accountID string) ([]Transaction, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/data/v1/accounts/%s/transactions", baseURL, accountID),
		nil,
	)
	if err != nil {
		return nil, err
	}
	c.authRequest(req)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	response := struct {
		Results []Transaction `json:"results"`
	}{}
	err = json.NewDecoder(res.Body).Decode(&response)
	sort.Slice(response.Results, func(i, j int) bool {
		return response.Results[i].Timestamp < response.Results[j].Timestamp
	})
	return response.Results, err
}

func (c *Client) Balance(ctx context.Context, accountID string) (*Balance, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/data/v1/accounts/%s/balance", baseURL, accountID),
		nil,
	)
	if err != nil {
		return nil, err
	}
	c.authRequest(req)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	response := struct {
		Results []Balance `json:"results"`
	}{}
	err = json.NewDecoder(res.Body).Decode(&response)
	if len(response.Results) != 1 {
		return nil, fmt.Errorf("unexpected length: %v", len(response.Results))
	}
	return &response.Results[0], err
}
