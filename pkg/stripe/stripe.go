package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/monzo/slog"

	"github.com/stripe/stripe-go/v71"
	portalsession "github.com/stripe/stripe-go/v71/billingportal/session"
	"github.com/stripe/stripe-go/v71/checkout/session"
	"github.com/stripe/stripe-go/v71/sub"
	"github.com/stripe/stripe-go/v71/webhook"

	"github.com/arussellsaw/youneedaspreadsheet/domain"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/authn"
	"github.com/arussellsaw/youneedaspreadsheet/pkg/util"
)

func Init(ctx context.Context, m *mux.Router) error {
	stripe.Key = os.Getenv("STRIPE_KEY")
	if stripe.Key == "" {
		return fmt.Errorf("missing stripe key")
	}

	m.HandleFunc("/api/stripe/setup", handleCreateCheckoutSession)
	m.HandleFunc("/api/stripe/success", handleSuccess)
	m.HandleFunc("/api/stripe/webhook", handleWebhook)
	m.HandleFunc("/api/stripe/portal", handleCustomerPortal)

	return nil
}

func HasSubscription(ctx context.Context, u *domain.User) (bool, error) {
	if u.Stripe.FreeForMyBuds {
		return true, nil
	}
	if u.Stripe.CustomerID == "" {
		return false, nil
	}
	if u.Stripe.PaidUntil.After(time.Now()) {
		slog.Debug(ctx, "paid up until %s", u.Stripe.PaidUntil)
		return true, nil
	}
	params := &stripe.SubscriptionListParams{
		Customer: u.Stripe.CustomerID,
		Price:    os.Getenv("PRICE_ID"),
	}
	i := sub.List(params)
	if i.Err() != nil {
		return false, i.Err()
	}
	for i.Next() {
		s := i.Subscription()
		if s.Status == stripe.SubscriptionStatusActive {
			u.Stripe.PaidUntil = time.Unix(s.CurrentPeriodEnd, 0)
			err := domain.UpdateUser(ctx, u)
			slog.Debug(ctx, "checked active subscription, paid up until %s", u.Stripe.PaidUntil)
			return true, err
		}
	}
	slog.Debug(ctx, "inactive subscription")
	return false, nil
}

func handleCreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	u := authn.User(r.Context())
	if u == nil {
		return
	}
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Price string `json:"priceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("json.NewDecoder.Decode: %v", err)
		return
	}

	// See https://stripe.com/docs/api/checkout/sessions/create
	// for additional parameters to pass.
	// {CHECKOUT_SESSION_ID} is a string literal; do not change it!
	// the actual Session ID is returned in the query parameter when your customer
	// is redirected to the success page.

	params := &stripe.CheckoutSessionParams{
		CustomerEmail: &u.Email,
		SuccessURL:    stripe.String(util.BaseURL() + "/api/stripe/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:     stripe.String(util.BaseURL() + "/"),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			&stripe.CheckoutSessionLineItemParams{
				Price: stripe.String(req.Price),
				// For metered billing, do not pass quantity
				Quantity: stripe.Int64(1),
			},
		},
	}

	s, err := session.New(params)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, struct {
			ErrorData string `json:"error"`
		}{
			ErrorData: "test",
		})
		return
	}

	writeJSON(w, struct {
		SessionID string `json:"sessionId"`
	}{
		SessionID: s.ID,
	})
}

func handleSuccess(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		return
	}
	ctx := r.Context()
	slog.Info(ctx, "payment success: %s", sessionID)
	u := authn.User(ctx)
	if u == nil {
		slog.Error(ctx, "got subscription success from unauthenticated user")
		return
	}
	u.Stripe.SessionID = sessionID
	s, err := session.Get(u.Stripe.SessionID, nil)
	if err != nil {
		slog.Error(ctx, "error getting session: %s", err)
		return
	}
	u.Stripe.CustomerID = s.Customer.ID
	u.Stripe.PaidUntil = time.Now().Add(32 * 24 * time.Hour)
	u.Stripe.Error = ""

	err = domain.UpdateUser(ctx, u)
	if err != nil {
		slog.Error(ctx, "error updating user: %s", err)
	}
	http.Redirect(w, r, "/", 302)
}

func writeJSON(w io.Writer, v interface{}) {
	json.NewEncoder(w).Encode(v)
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("ioutil.ReadAll: %v", err)
		return
	}

	event, err := webhook.ConstructEvent(b, r.Header.Get("Stripe-Signature"), os.Getenv("STRIPE_WEBHOOK_SECRET"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("webhook.ConstructEvent: %v", err)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		slog.Info(ctx, "checkout completed: %s", event.ID)
		// Payment is successful and the subscription is created.
		// You should provision the subscription.
	case "invoice.paid":
		slog.Info(ctx, "invoice paid: %s", event.ID)
		// Continue to provision the subscription as payments continue to be made.
		// Store the status in your database and check when a user accesses your service.
		// This approach helps you avoid hitting rate limits.
	case "invoice.payment_failed":
		slog.Error(ctx, "invoice payment failed: %s", event.ID)
		// The payment failed or the customer does not have a valid payment method.
		// The subscription becomes past_due. Notify your customer and send them to the
		// customer portal to update their payment information.
	default:
		slog.Info(ctx, "unhandled event type: %s", event.Type)
		// unhandled event type
	}
}

func handleCustomerPortal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("json.NewDecoder.Decode: %v", err)
		return
	}

	// For demonstration purposes, we're using the Checkout session to retrieve the customer ID.
	// Typically this is stored alongside the authenticated user in your database.
	sessionID := req.SessionID
	s, err := session.Get(sessionID, nil)
	if err != nil {
		slog.Error(ctx, "error getting session: %s", err)
		return
	}

	// The URL to which the user is redirected when they are done managing
	// billing in the portal.
	returnURL := util.BaseURL() + "/"

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(s.Customer.ID),
		ReturnURL: stripe.String(returnURL),
	}
	ps, err := portalsession.New(params)
	if err != nil {
		slog.Error(ctx, "error getting session: %s", err)
		return
	}

	writeJSON(w, struct {
		URL string `json:"url"`
	}{
		URL: ps.URL,
	})
}
