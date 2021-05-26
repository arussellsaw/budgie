package handler

import (
	"html/template"
	"net/http"
	"os"

	"github.com/monzo/slog"

	"github.com/arussellsaw/budgie/pkg/authn"
)

func handleBusiness(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	t := template.New("business.html")
	t, err := t.ParseFiles("tmpl/business.html")
	if err != nil {
		slog.Error(ctx, "Error parsing template: %s", err)
		http.Error(w, err.Error(), 500)
		return
	}
	u := authn.User(ctx)
	hasTL, accs := hasTruelayer(ctx, u)
	hasGS := hasSheets(ctx, u)
	hasS := hasStripe(ctx, u)
	err = t.Execute(w, indexData{
		User:                 u,
		HasTruelayer:         hasTL,
		HasSheets:            hasGS,
		HasStripe:            hasS,
		Accounts:             accs,
		StripePublishableKey: os.Getenv("STRIPE_PUBLISHABLE_KEY"),
		StripePriceID:        os.Getenv("STRIPE_BUSINESS_PRICE_ID"),
	})
	if err != nil {
		slog.Error(ctx, "Index: %s", err)
	}
}
