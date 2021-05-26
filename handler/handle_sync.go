package handler

import (
	"encoding/json"
	"hash/fnv"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/monzo/slog"
	"golang.org/x/time/rate"
	gsheets "google.golang.org/api/sheets/v4"

	"github.com/arussellsaw/budgie/domain"
	"github.com/arussellsaw/budgie/pkg/authn"
	"github.com/arussellsaw/budgie/pkg/logging"
	"github.com/arussellsaw/budgie/pkg/sheets"
	"github.com/arussellsaw/budgie/pkg/stripe"
	"github.com/arussellsaw/budgie/pkg/truelayer"
)

var (
	limiter = rate.NewLimiter(rate.Every(30*time.Second), 1)
)

type pubSubMessage struct {
	Message      pubsub.Message `json:"message"`
	Subscription string         `json:"subscription"`
}

func handleSync(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		u        = authn.User(ctx)
		err      error
		errs     []error
		historic bool
	)
	if u == nil {
		m := pubSubMessage{}
		err = json.NewDecoder(r.Body).Decode(&m)
		if err != nil {
			slog.Error(ctx, "error decoding: %s", err)
			return
		}
		data := string(m.Message.Data)
		parts := strings.Split(data, "|")
		userID := parts[0]
		if len(parts) == 2 {
			historic, _ = strconv.ParseBool(parts[1])
		}
		u, err = domain.UserByID(ctx, userID)
		if err != nil {
			slog.Error(ctx, "error getting user: %s", err)
			return
		}
	}
	ctx = logging.WithParams(ctx, map[string]string{"user_id": u.ID})

	if !limiter.Allow() && r.Method == http.MethodPost {
		slog.Warn(ctx, "rate limit exceeded, backing off: %s", u.ID)
		http.Error(w, "rate limit exceeded", 429)
		return
	}

	defer func() {
		if len(errs) == 0 {
			slog.Info(ctx, "user %s sync complete", u.ID)
		} else {
			slog.Warn(ctx, "user %s sync complete, with warnings: %s", u.ID, errs)
		}
	}()

	slog.Info(ctx, "sync user: %s", u.ID)
	if time.Since(u.LastSync) < 5*time.Minute {
		slog.Info(ctx, "skipping user %s, last synced at %s", u.ID, u.LastSync)
		return
	}

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
		slog.Warn(ctx, "Error getting truelayer client: %s", err)
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
		as, aerr := tl.Accounts(ctx)
		if aerr != nil {
			slog.Warn(ctx, "Error getting accounts: %s", aerr)
		}
		for _, a := range as {
			a := a
			accs = append(accs, a)
		}
		cs, cerr := tl.Cards(ctx)
		if cerr != nil {
			slog.Warn(ctx, "Error getting cards: %s", cerr)
		}
		for _, c := range cs {
			c := c
			accs = append(accs, c)
		}
		if len(cs) == 0 && len(as) == 0 {
			errs = append(errs, aerr, cerr)
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
		select {
		case <-ctx.Done():
			slog.Error(ctx, "context timeout")
			return
		default:
		}
		ctx := logging.WithParams(ctx, map[string]string{
			"token_id":   acc.TokenID(),
			"account_id": acc.ID(),
			"user_id":    u.ID,
		})
		slog.Info(ctx, "syncing account %s", acc.ID())
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
		txs, err := acc.Transactions(ctx, historic)
		if err != nil {
			slog.Warn(ctx, "Error getting transactions: %s", err)
			errs = append(errs, err)
			continue
		}
		slog.Debug(ctx, "got %v transactions", len(txs))
		sort.Slice(txs, func(i, j int) bool {
			return txs[i].Timestamp < txs[j].Timestamp
		})
		update := buildUpdate(txs, accSheet)
		if update == nil {
			slog.Info(ctx, "skipping empty update for account %s", acc.ID())
			continue
		}
		slog.Info(ctx, "successfully synced account %s", acc.ID())
		reqs = append(reqs, update...)

	}

	var balances []truelayer.Balance
	for _, acc := range accs {
		ctx := logging.WithParams(ctx, map[string]string{
			"token_id":   acc.TokenID(),
			"account_id": acc.ID(),
			"user_id":    u.ID,
		})
		b, err := acc.Balance(ctx)
		if err != nil {
			slog.Warn(ctx, "error getting balance: %s", err)
			errs = append(errs, err)
			continue
		}
		balances = append(balances, *b)
	}
	if len(balances) != 0 && balanceSheet != nil {
		reqs = append(reqs, balanceUpdate(accs, balances, balanceSheet))
	}

	_, err = gs.Service(ctx).Spreadsheets.BatchUpdate(u.SheetID, &gsheets.BatchUpdateSpreadsheetRequest{
		Requests: reqs,
	}).Context(ctx).Do()
	if err != nil {
		slog.Error(ctx, "Error updating sheet %s : %s", u.ID, err)
		return
	}

	u.LastSync = time.Now().UTC()
	err = domain.UpdateUser(ctx, u)
	if err != nil {
		slog.Warn(ctx, "Error updating last sync time: %s", err)
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
