package logging

import (
	"context"
	"fmt"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/authn"

	"github.com/arussellsaw/youneedaspreadsheet/pkg/util"

	"cloud.google.com/go/errorreporting"
	"github.com/monzo/slog"
)

var _ slog.Logger = &ReportingLogger{}

func NewReportingLogger(ctx context.Context, logger slog.Logger) (slog.Logger, error) {
	client, err := errorreporting.NewClient(ctx, util.Project(), errorreporting.Config{
		ServiceName: "youneedaspreadsheet",
		OnError: func(err error) {
			slog.Warn(ctx, "error reporting error: %s", err)
		},
	})
	return &ReportingLogger{
		Logger: logger,
		cl:     client,
	}, err
}

type ReportingLogger struct {
	slog.Logger
	cl *errorreporting.Client
}

func (l *ReportingLogger) Log(evs ...slog.Event) {
	for _, e := range evs {
		if e.Severity >= slog.ErrorSeverity {
			u := authn.User(e.Context)
			userID, _ := e.Labels["user_id"]
			l.cl.Report(errorreporting.Entry{
				Error: fmt.Errorf("%s", e.Message),
				User: func() string {
					if userID != "" {
						return userID
					}
					if u != nil {
						return u.ID
					}
					return ""
				}(),
			})
		}
	}
	l.Logger.Log(evs...)
}
