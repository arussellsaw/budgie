package sheets

import (
	"context"

	"golang.org/x/oauth2"

	"github.com/pkg/errors"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/token"
)

var ErrSheetNotFound = errors.New("not_found.sheet: couldn't find sheet in context")

type Client struct {
	c *sheets.Service
}

func NewClient(ctx context.Context, userID string) (*Client, error) {
	t, err := token.Get(ctx, OauthConfig, userID)
	if err != nil {
		return nil, errors.Wrap(err, "getting token")
	}
	src := oauth2.StaticTokenSource(t)

	srv, err := sheets.NewService(
		ctx,
		option.WithTokenSource(src),
	)
	if err != nil {
		return nil, errors.Wrap(err, "getting service")
	}

	return &Client{
		c: srv,
	}, nil
}

func (c *Client) Create(ctx context.Context) (string, error) {
	res, err := c.c.Spreadsheets.Create(&sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: "Banksheets Export",
		},
	}).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return res.SpreadsheetId, nil
}

func (c *Client) Service(ctx context.Context) *sheets.Service {
	return c.c
}

func (c *Client) Get(ctx context.Context, sheetID string) (*sheets.Spreadsheet, error) {
	return c.c.Spreadsheets.Get(sheetID).IncludeGridData(true).Context(ctx).Do()
}
