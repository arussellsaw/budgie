package logging

import (
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/monzo/slog"
)

type ColourLogger struct {
	Writer io.Writer
}

func (l ColourLogger) Log(evs ...slog.Event) {
	for _, e := range evs {
		switch e.Severity {
		case slog.TraceSeverity:
			fmt.Fprintf(l.Writer, "%s: %s\n", color.WhiteString("%s TRC", e.Timestamp.Format("15:04:05.000")), e.Message)
		case slog.DebugSeverity:
			fmt.Fprintf(l.Writer, "%s: %s\n", color.CyanString("%s DBG", e.Timestamp.Format("15:04:05.000")), e.Message)
		case slog.InfoSeverity:
			fmt.Fprintf(l.Writer, "%s: %s\n", color.BlueString("%s INF", e.Timestamp.Format("15:04:05.000")), e.Message)
		case slog.WarnSeverity:
			fmt.Fprintf(l.Writer, "%s: %s\n", color.YellowString("%s WRN", e.Timestamp.Format("15:04:05.000")), e.Message)
		case slog.ErrorSeverity:
			fmt.Fprintf(l.Writer, "%s: %s\n", color.RedString("%s ERR", e.Timestamp.Format("15:04:05.000")), e.Message)
		case slog.CriticalSeverity:
			fmt.Fprintf(l.Writer, "%s: %s\n", color.RedString("%s CRT", e.Timestamp.Format("15:04:05.000")), e.Message)
		}
	}
}

func (l ColourLogger) Flush() error { return nil }
