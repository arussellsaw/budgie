package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/arussellsaw/bank-sheets/pkg/sheets"

	"github.com/arussellsaw/bank-sheets/pkg/truelayer"

	"github.com/monzo/slog"

	"github.com/arussellsaw/bank-sheets/domain"

	gsheets "google.golang.org/api/sheets/v4"
)

func handleSync(w http.ResponseWriter, r *http.Request) {
	var (
		ctx = r.Context()
		u   = domain.UserFromContext(ctx)
		err error
	)
	if u == nil {
		userID := r.URL.Query().Get("user_id")
		u, err = domain.UserByID(ctx, userID)
		if err != nil {
			slog.Error(ctx, "Error getting user: %s", err)
			return
		}
	}
	if u.SheetID == "" {
		slog.Error(ctx, "No sheet ID for user %s", u.ID)
		return
	}
	tl, err := truelayer.NewClient(ctx, u.ID)
	if err != nil {
		slog.Error(ctx, "Error getting truelayer client: %s", err)
		return
	}
	gs, err := sheets.NewClient(ctx, u.ID)
	if err != nil {
		slog.Error(ctx, "Error getting sheets client: %s", err)
		return
	}
	accs, err := tl.Accounts(ctx)
	if err != nil {
		slog.Error(ctx, "Error getting accounts: %s", err)
		return
	}
	userSheet, err := gs.Get(ctx, u.SheetID)
	if err != nil {
		slog.Error(ctx, "Error getting sheet: %s", err)
		return
	}
	for _, acc := range accs {
		attempted := false
	findSheet:
		var accSheet *gsheets.Sheet
		for _, sheet := range userSheet.Sheets {
			if sheet.Properties.Title == acc.DisplayName {
				accSheet = sheet
			}
		}
		if accSheet == nil {
			if attempted {
				slog.Error(ctx, "failed to modify sheets")
				return
			}
			_, err = gs.Service(ctx).Spreadsheets.BatchUpdate(u.SheetID, &gsheets.BatchUpdateSpreadsheetRequest{
				Requests: []*gsheets.Request{
					{
						AddSheet: &gsheets.AddSheetRequest{
							Properties: &gsheets.SheetProperties{
								Title: acc.DisplayName,
							},
						},
					},
				},
			}).Context(ctx).Do()
			if err != nil {
				slog.Error(ctx, "Error adding new sheet: %s", err)
				return
			}
			userSheet, err = gs.Get(ctx, u.SheetID)
			if err != nil {
				slog.Error(ctx, "Error getting sheet: %s", err)
				return
			}
			attempted = true
			goto findSheet
		}
		txs, err := tl.Transactions(ctx, acc.AccountID)
		if err != nil {
			slog.Error(ctx, "Error getting transactions: %s", err)
			return
		}
		update := buildUpdate(u.SheetID, txs, accSheet)
		if update == nil {
			continue
		}
		_, err = gs.Service(ctx).Spreadsheets.BatchUpdate(u.SheetID, update).Context(ctx).Do()
		if err != nil {
			slog.Error(ctx, "Error building update: %s", err)
			return
		}

	}
	u.LastSync = time.Now()
	err = domain.UpdateUser(ctx, u)
	if err != nil {
		slog.Error(ctx, "Error updating last sync time: %s", err)
	}
}

func buildUpdate(sheetID string, txs []truelayer.Transaction, sheet *gsheets.Sheet) *gsheets.BatchUpdateSpreadsheetRequest {
	if len(sheet.Data) == 0 {
		return nil
	}
	existing := make(map[string]struct{})
	for _, row := range sheet.Data[0].RowData {
		txid := *row.Values[0].UserEnteredValue.StringValue
		existing[txid] = struct{}{}
	}
	filtered := []truelayer.Transaction{}
	for _, tx := range txs {
		if _, ok := existing[tx.TransactionID]; ok {
			continue
		}
		filtered = append(filtered, tx)
	}
	if len(filtered) == 0 {
		slog.Debug(context.Background(), "empty sync")
		return nil
	}
	return &gsheets.BatchUpdateSpreadsheetRequest{
		Requests: []*gsheets.Request{
			{
				AppendCells: &gsheets.AppendCellsRequest{
					Fields:  "*",
					SheetId: sheet.Properties.SheetId,
					Rows:    buildRows(filtered),
				},
			},
		},
	}
}

func buildRows(txs []truelayer.Transaction) []*gsheets.RowData {
	rows := []*gsheets.RowData{}
	for _, tx := range txs {
		tx := tx
		rd := gsheets.RowData{
			Values: []*gsheets.CellData{
				{
					UserEnteredValue: &gsheets.ExtendedValue{
						StringValue: &tx.TransactionID,
					},
				},
				{
					UserEnteredValue: &gsheets.ExtendedValue{
						StringValue: &tx.Timestamp,
					},
				},
				{
					UserEnteredValue: &gsheets.ExtendedValue{
						NumberValue: &tx.Amount,
					},
				},
				{
					UserEnteredValue: &gsheets.ExtendedValue{
						StringValue: &tx.Currency,
					},
				},
				{
					UserEnteredValue: &gsheets.ExtendedValue{
						StringValue: &tx.MerchantName,
					},
				},
				{
					UserEnteredValue: &gsheets.ExtendedValue{
						StringValue: &tx.TransactionCategory,
					},
				},
				{
					UserEnteredValue: &gsheets.ExtendedValue{
						NumberValue: &tx.RunningBalance.Amount,
					},
				},
			},
		}
		rows = append(rows, &rd)
	}
	return rows
}
