package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/blackswarm/hive/internal/protocol"
)

// GenerateReport creates a Markdown report containing agent inventory,
// tasks dispatched, and results received.
func GenerateReport(manager *AgentManager, tasks []protocol.TaskMsg, results []protocol.ResultMsg) (string, error) {
	var b strings.Builder

	b.WriteString("# Hive Operation Report\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339)))

	// Agent list
	b.WriteString("## Agents\n\n")
	b.WriteString("| ID | Hostname | OS | IP | Last Seen | Status |\n")
	b.WriteString("|----|----------|----|----|-----------|--------|\n")
	for _, a := range manager.ListAgents() {
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n",
			a.ID, a.Hostname, a.OS, a.IP,
			a.LastSeen.Format(time.RFC3339),
			a.Status))
	}
	b.WriteString("\n")

	// Tasks
	b.WriteString("## Tasks\n\n")
	b.WriteString("| Task ID | Command | Timeout (s) |\n")
	b.WriteString("|---------|---------|-------------|\n")
	for _, t := range tasks {
		timeout := t.Timeout
		if timeout == 0 {
			timeout = 60 // default
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %d |\n", t.TaskID, t.Command, timeout))
	}
	b.WriteString("\n")

	// Results
	b.WriteString("## Results\n\n")
	if len(results) == 0 {
		b.WriteString("*No results yet.*\n\n")
	} else {
		b.WriteString("| Task ID | Stdout | Stderr | Error |\n")
		b.WriteString("|---------|--------|--------|-------|\n")
		for _, r := range results {
			stdout := strings.ReplaceAll(r.Stdout, "\n", "\\n")
			stderr := strings.ReplaceAll(r.Stderr, "\n", "\\n")
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				r.TaskID, stdout, stderr, r.Error))
		}
		b.WriteString("\n")
	}

	return b.String(), nil
}
