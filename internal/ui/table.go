package ui

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	ghwt "github.com/hanenao/git-handoff/internal/worktree"
)

func RenderWorktreeTable(rows []ghwt.ListRow) (string, error) {
	var builder strings.Builder
	writer := tabwriter.NewWriter(&builder, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "CURRENT\tKIND\tWORKTREE\tSTATE\tBRANCH\tPATH\tUPDATED"); err != nil {
		return "", err
	}
	for _, row := range rows {
		current := ""
		if row.IsCurrent {
			current = "*"
		}
		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			current,
			row.Kind,
			row.ID,
			row.State,
			row.Branch,
			row.Path,
			formatTime(row.UpdatedAt),
		); err != nil {
			return "", err
		}
	}
	if err := writer.Flush(); err != nil {
		return "", err
	}
	return builder.String(), nil
}

func WriteString(w io.Writer, value string) error {
	_, err := io.WriteString(w, value)
	return err
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Local().Format(time.RFC3339)
}
