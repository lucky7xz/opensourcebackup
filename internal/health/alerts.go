package health

// Alert is a single operational alert derived from a health score deduction.
// Alerts are stateless — they are computed fresh on each request.
// No persistence needed for MVP: if the condition clears, the alert disappears.
type Alert struct {
	Code        string `json:"code"`        // machine-readable, matches Deduction.Code
	Severity    string `json:"severity"`    // critical | warning | info
	Category    string `json:"category"`    // backup | restore | agent | repository | retention
	Title       string `json:"title"`       // short, action-oriented
	Description string `json:"description"` // full explanation from score deduction
	Points      int    `json:"points"`      // score impact
	Action      string `json:"action"`      // what the operator should do
}

// alertMeta maps deduction codes to enriched alert metadata.
// All fields except Description (which comes from the deduction) are static.
var alertMeta = map[string]struct {
	severity string
	category string
	title    string
	action   string
}{
	"no_restore_tests": {
		severity: "critical",
		category: "restore",
		title:    "No restore tests configured",
		action:   "Go to Restore Tests → schedule a restore test for your most important snapshots",
	},
	"partial_restore_coverage": {
		severity: "warning",
		category: "restore",
		title:    "Some snapshots are not restore-tested",
		action:   "Go to Restore Tests → run restore tests for untested snapshots",
	},
	"restore_test_stale": {
		severity: "warning",
		category: "restore",
		title:    "Restore test older than 30 days",
		action:   "Schedule a new restore test to confirm recoverability",
	},
	"no_successful_backup": {
		severity: "critical",
		category: "backup",
		title:    "No successful backup recorded",
		action:   "Check agent connectivity and run a manual backup job",
	},
	"backup_stale_24h": {
		severity: "warning",
		category: "backup",
		title:    "No backup in the last 24 hours",
		action:   "Check if the scheduled backup policy ran and agent is online",
	},
	"failed_jobs_24h": {
		severity: "warning",
		category: "backup",
		title:    "Backup jobs failed in last 24 hours",
		action:   "Go to Jobs → check failed job error messages",
	},
	"high_failure_rate": {
		severity: "warning",
		category: "backup",
		title:    "High backup failure rate (>20%)",
		action:   "Review job history and check storage connectivity",
	},
	"agents_offline": {
		severity: "critical",
		category: "agent",
		title:    "One or more agents are offline",
		action:   "Check agent service status: opensourcebackup-agent status",
	},
	"agents_idle": {
		severity: "info",
		category: "agent",
		title:    "One or more agents are idle",
		action:   "Agent has not sent a heartbeat for 2–15 minutes — may be recovering",
	},
	"repos_not_immutable": {
		severity: "warning",
		category: "repository",
		title:    "Repositories without write protection",
		action:   "Go to Repositories → set immutable_mode to object_lock, worm, or append_only",
	},
	"repos_not_encrypted": {
		severity: "warning",
		category: "repository",
		title:    "Repositories without encryption",
		action:   "Go to Repositories → configure an encryption mode (e.g. aes256)",
	},
	"no_retention_policy": {
		severity: "info",
		category: "retention",
		title:    "No retention rules configured",
		action:   "Go to Policies → set keep_last or keep_daily to prevent unbounded growth",
	},
}

// AlertsFromScore converts health score deductions into operational alerts.
// Alerts are ordered by severity: critical → warning → info.
func AlertsFromScore(result ScoreResult) []Alert {
	var critical, warning, info []Alert

	for _, d := range result.Deductions {
		meta, ok := alertMeta[d.Code]
		if !ok {
			// Unknown deduction — emit as info
			warning = append(warning, Alert{
				Code:        d.Code,
				Severity:    "info",
				Category:    "system",
				Title:       d.Code,
				Description: d.Reason,
				Points:      d.Points,
			})
			continue
		}
		a := Alert{
			Code:        d.Code,
			Severity:    meta.severity,
			Category:    meta.category,
			Title:       meta.title,
			Description: d.Reason,
			Points:      d.Points,
			Action:      meta.action,
		}
		switch meta.severity {
		case "critical":
			critical = append(critical, a)
		case "warning":
			warning = append(warning, a)
		default:
			info = append(info, a)
		}
	}

	all := append(critical, warning...)
	return append(all, info...)
}

// AlertSummary provides aggregate counts for dashboard display.
type AlertSummary struct {
	Total    int `json:"total"`
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
}

// Summarize counts alerts by severity.
func Summarize(alerts []Alert) AlertSummary {
	s := AlertSummary{Total: len(alerts)}
	for _, a := range alerts {
		switch a.Severity {
		case "critical":
			s.Critical++
		case "warning":
			s.Warning++
		default:
			s.Info++
		}
	}
	return s
}
