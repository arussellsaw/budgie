package handler

import (
	"encoding/json"
	"hash/fnv"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/logging"

	"cloud.google.com/go/pubsub"
	"github.com/monzo/slog"
	gsheets "google.golang.org/api/sheets/v4"

	"github.com/arussellsaw/youneedaspreadsheet/domain"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/authn"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/sheets"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/stripe"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/truelayer"
)

type pubSubMessage struct {
	Message      pubsub.Message `json:"message"`
	Subscription string         `json:"subscription"`
}

func handleSync(w http.ResponseWriter, r *http.Request) {
	var (
		ctx = r.Context()
		u   = authn.User(ctx)
		err error
	)
	if u == nil {
		m := pubSubMessage{}
		err = json.NewDecoder(r.Body).Decode(&m)
		if err != nil {
			slog.Error(ctx, "error decoding: %s", err)
			return
		}
		userID := string(m.Message.Data)
		u, err = domain.UserByID(ctx, userID)
		if err != nil {
			slog.Error(ctx, "error getting user: %s", err)
			return
		}
	}
	ctx = logging.WithParams(ctx, map[string]string{"user_id": u.ID})

	slog.Info(ctx, "sync user: %s", u.ID)

	if u.SheetID == "" {
		slog.Error(ctx, "No sheet ID for user %s", u.ID)
		http.Error(w, "You need to set up a sheet, go back to the homepage", http.StatusBadRequest)
		return
	}
	ok, err := stripe.HasSubscription(ctx, u)
	if err != nil || !ok {
		slog.Error(ctx, "error checking for subscription: %s", err)
		http.Error(w, "You need to set up your stripe subscription, go back to the homepage", http.StatusForbidden)
		return
	}
	tls, err := truelayer.GetClients(ctx, u.ID)
	if err != nil {
		slog.Error(ctx, "Error getting truelayer client: %s", err)
		if len(tls) == 0 {
			slog.Error(ctx, "UNABLE TO SYNC USER, NO TRUELAYER CLIENTS %s", u.ID)
			return
		}
	}
	gs, err := sheets.NewClient(ctx, u.ID)
	if err != nil {
		slog.Error(ctx, "Error getting sheets client: %s", err)
		return
	}
	var accs []truelayer.AbstractAccount
	for _, tl := range tls {
		as, err := tl.Accounts(ctx)
		if err != nil {
			slog.Error(ctx, "Error getting accounts: %s", err)
			return
		}
		for _, a := range as {
			a := a
			accs = append(accs, a)
		}
		cs, err := tl.Cards(ctx)
		if err != nil {
			slog.Error(ctx, "Error getting cards: %s", err)
		}
		for _, c := range cs {
			c := c
			accs = append(accs, c)
		}
	}
	userSheet, err := gs.Get(ctx, u.SheetID)
	if err != nil {
		slog.Error(ctx, "Error getting sheet: %s", err)
		return
	}
	var (
		reqs         []*gsheets.Request
		balanceSheet *gsheets.Sheet
	)
	for _, acc := range accs {
		attempted := false
	findSheet:
		var accSheet *gsheets.Sheet
		for _, sheet := range userSheet.Sheets {
			if sheet.Properties.SheetId == sheetID(acc.ID()) {
				accSheet = sheet
			}
			if strings.HasPrefix(sheet.Properties.Title, acc.Name()) {
				if len(sheet.Properties.Title) > len(acc.Name()) && !strings.HasSuffix(sheet.Properties.Title, acc.ID()) {
					continue
				}
				accSheet = sheet
			}
			if sheet.Properties.Title == "Sheet1" {
				balanceSheet = sheet
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
								SheetId: sheetID(acc.ID()),
								Title:   acc.Name(),
								GridProperties: &gsheets.GridProperties{
									ColumnCount: 7,
									RowCount:    5,
								},
							},
						},
					},
				},
			}).Context(ctx).Do()
			if err != nil {
				slog.Error(ctx, "Error adding new sheet %s: %s", u.ID, err)
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
		txs, err := acc.Transactions(ctx, true)
		if err != nil {
			slog.Error(ctx, "Error getting transactions: %s", err)
			return
		}
		sort.Slice(txs, func(i, j int) bool {
			return txs[i].Timestamp < txs[j].Timestamp
		})
		update := buildUpdate(txs, accSheet)
		if update == nil {
			continue
		}
		reqs = append(reqs, update...)

	}
	u.LastSync = time.Now()
	err = domain.UpdateUser(ctx, u)
	if err != nil {
		slog.Error(ctx, "Error updating last sync time: %s", err)
	}

	var balances []truelayer.Balance
	for _, acc := range accs {
		b, err := acc.Balance(ctx)
		if err != nil {
			slog.Error(ctx, "error getting balance: %s", err)
			return
		}
		balances = append(balances, *b)
	}
	reqs = append(reqs, balanceUpdate(accs, balances, balanceSheet))

	_, err = gs.Service(ctx).Spreadsheets.BatchUpdate(u.SheetID, &gsheets.BatchUpdateSpreadsheetRequest{
		Requests: reqs,
	}).Context(ctx).Do()
	if err != nil {
		slog.Error(ctx, "Error updating sheet %s : %s", u.ID, err)
		return
	}
	if r.Method == http.MethodGet {
		http.Redirect(w, r, "/", 302)
	}
}

func buildUpdate(txs []truelayer.Transaction, sheet *gsheets.Sheet) []*gsheets.Request {
	if len(sheet.Data) == 0 {
		return nil
	}
	existing := make(map[string]struct{})
	for _, row := range sheet.Data[0].RowData {
		if row == nil || len(row.Values) == 0 || row.Values[0] == nil || row.Values[0].UserEnteredValue == nil || row.Values[0].UserEnteredValue.StringValue == nil {
			continue
		}
		txid := *row.Values[0].UserEnteredValue.StringValue
		existing[txid] = struct{}{}
	}
	rows := buildRows(txs, sheet.Data[0].RowData)
	var reqs []*gsheets.Request
	if len(rows) > len(existing) {
		reqs = append(reqs, &gsheets.Request{
			UpdateCells: &gsheets.UpdateCellsRequest{
				Fields: "*",
				Range: &gsheets.GridRange{
					SheetId:          sheet.Properties.SheetId,
					StartRowIndex:    0,
					StartColumnIndex: 0,
					EndColumnIndex:   0,
					EndRowIndex:      0,
				},
				Rows: rows[:len(existing)],
			},
		})
		reqs = append(reqs, &gsheets.Request{
			AppendCells: &gsheets.AppendCellsRequest{
				SheetId: sheet.Properties.SheetId,
				Fields:  "*",
				Rows:    rows[len(existing):],
			},
		})
		return reqs
	}
	return []*gsheets.Request{
		{
			UpdateCells: &gsheets.UpdateCellsRequest{
				Fields: "*",
				Range: &gsheets.GridRange{
					SheetId:          sheet.Properties.SheetId,
					StartRowIndex:    0,
					StartColumnIndex: 0,
					EndColumnIndex:   0,
					EndRowIndex:      0,
				},
				Rows: rows,
			},
		},
	}
}

func buildRows(txs []truelayer.Transaction, existing []*gsheets.RowData) []*gsheets.RowData {
	rows := []*gsheets.RowData{}
	newRecs := make(map[string]struct{})
	for _, tx := range txs {
		tx := tx
		newRecs[tx.TransactionID] = struct{}{}
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
						StringValue: &tx.Description,
					},
				},
			},
		}
		rows = append(rows, &rd)
	}
	for _, rd := range existing {
		if rd.Values == nil || len(rd.Values) == 0 || rd.Values[0].UserEnteredValue == nil || rd.Values[0].UserEnteredValue.StringValue == nil {
			continue
		}
		if sv := rd.Values[0].UserEnteredValue.StringValue; sv != nil {
			if _, ok := newRecs[*sv]; ok {
				continue
			}
		}
		rows = append(rows, rd)
	}
	sort.Slice(rows, func(i, j int) bool {
		return timestamp(rows[i]) < timestamp(rows[j])
	})
	return rows
}

func timestamp(row *gsheets.RowData) string {
	if row == nil || len(row.Values) < 1 || row.Values[1].UserEnteredValue == nil || row.Values[1].UserEnteredValue.StringValue == nil {
		return ""
	}
	return *row.Values[1].UserEnteredValue.StringValue
}

func balanceUpdate(accs []truelayer.AbstractAccount, balances []truelayer.Balance, sheet *gsheets.Sheet) *gsheets.Request {
	return &gsheets.Request{
		UpdateCells: &gsheets.UpdateCellsRequest{
			Fields: "*",
			Range: &gsheets.GridRange{
				SheetId:          sheet.Properties.SheetId,
				StartRowIndex:    0,
				StartColumnIndex: 0,
				EndColumnIndex:   0,
				EndRowIndex:      0,
			},
			Rows: func() []*gsheets.RowData {
				rows := []*gsheets.RowData{}
				rows = append(rows, &gsheets.RowData{
					Values: []*gsheets.CellData{
						{
							UserEnteredValue: &gsheets.ExtendedValue{
								StringValue: strPtr("Account"),
							},
						},
						{
							UserEnteredValue: &gsheets.ExtendedValue{
								StringValue: strPtr("Currency"),
							},
						},
						{
							UserEnteredValue: &gsheets.ExtendedValue{
								StringValue: strPtr("Available Balance"),
							},
						},
						{
							UserEnteredValue: &gsheets.ExtendedValue{
								StringValue: strPtr("Current Balance"),
							},
						},
						{
							UserEnteredValue: &gsheets.ExtendedValue{
								StringValue: strPtr("Provider"),
							},
						},
					},
				})
				for i, b := range balances {
					b := b
					rows = append(rows, &gsheets.RowData{
						Values: []*gsheets.CellData{
							{
								UserEnteredValue: &gsheets.ExtendedValue{
									StringValue: strPtr(accs[i].Name()),
								},
							},
							{
								UserEnteredValue: &gsheets.ExtendedValue{
									StringValue: &b.Currency,
								},
							},
							{
								UserEnteredValue: &gsheets.ExtendedValue{
									NumberValue: &b.Available,
								},
							},
							{
								UserEnteredValue: &gsheets.ExtendedValue{
									NumberValue: &b.Current,
								},
							},
							{
								UserEnteredValue: &gsheets.ExtendedValue{
									StringValue: strPtr(accs[i].ProviderName()),
								},
							},
						},
					})
				}
				return rows
			}(),
		},
	}
}

func sheetID(id string) int64 {
	h := fnv.New32()
	h.Write([]byte(id))
	return int64(h.Sum32() % 100000)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
