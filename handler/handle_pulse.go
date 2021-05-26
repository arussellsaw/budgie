package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/monzo/slog"

	gsheets "google.golang.org/api/sheets/v4"

	"github.com/arussellsaw/budgie/pkg/truelayer"

	"github.com/pkg/errors"

	"github.com/arussellsaw/budgie/pkg/sheets"

	"github.com/arussellsaw/budgie/pkg/util"

	"github.com/arussellsaw/budgie/pkg/authn"
)

type pulseData struct {
	Accounts     []truelayer.Account                `json:"accounts"`
	Transactions map[string][]truelayer.Transaction `json:"transactions"`
}

func handlePulse(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	u := authn.User(ctx)
	if u == nil {
		return util.WrapCode(fmt.Errorf("unauthorized"), 401)
	}

	tls, err := truelayer.GetClients(ctx, u.ID)
	if err != nil {
		return err
	}
	accs := []truelayer.Account{}
	for _, tl := range tls {
		res, err := tl.Accounts(ctx)
		if err != nil {
			return util.WrapCode(errors.Wrap(err, "getting accounts"), 500)
		}
		accs = append(accs, res...)
	}

	gs, err := sheets.NewClient(ctx, u.ID)
	if err != nil {
		return errors.Wrap(err, "Error getting sheets client")
	}

	userSheet, err := gs.Get(ctx, u.SheetID)
	if err != nil {
		return err
	}

	accTxs := make(map[string][]truelayer.Transaction)
	for _, acc := range accs {
		var txs []truelayer.Transaction
		for _, sheet := range userSheet.Sheets {
			if sheet.Properties.SheetId != sheetID(acc.AccountID) {
				continue
			}
			for _, d := range sheet.Data {
				for _, rd := range d.RowData {
					txn, err := parseTransaction(rd)
					if err != nil {
						slog.Warn(ctx, "error parsing transaction: %s", err)
						continue
					}
					txs = append(txs, *txn)
				}
			}
			accTxs[acc.AccountID] = txs
		}
	}

	w.Header().Add("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(pulseData{
		Accounts:     accs,
		Transactions: accTxs,
	})
}

func parseTransaction(data *gsheets.RowData) (*truelayer.Transaction, error) {
	if data == nil || len(data.Values) < 5 {
		return nil, fmt.Errorf("too few columns")
	}
	amount, err := strconv.ParseFloat(data.Values[2].FormattedValue, 64)
	if err != nil {
		return nil, err
	}
	return &truelayer.Transaction{
		TransactionID: data.Values[0].FormattedValue,
		Timestamp:     data.Values[1].FormattedValue,
		Amount:        amount,
		Currency:      data.Values[3].FormattedValue,
		Description:   data.Values[4].FormattedValue,
	}, nil
}
