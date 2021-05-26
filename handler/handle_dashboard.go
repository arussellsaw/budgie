package handler

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/pkg/errors"

	"github.com/arussellsaw/budgie/pkg/authn"
	"github.com/arussellsaw/budgie/pkg/util"
)

func handleDashboard(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	t := template.New("dashboard.html")
	t, err := t.ParseFiles("tmpl/dashboard.html")
	if err != nil {
		return errors.Wrap(err, "parsing template")
	}

	u := authn.User(ctx)
	if u == nil {
		return util.WrapCode(fmt.Errorf("not authorised"), 401)
	}

	return t.Execute(w, nil)
}
