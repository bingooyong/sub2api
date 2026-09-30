package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func (r *scheduledTestPlanRepository) ListQualityPlans(ctx context.Context) ([]*service.ScheduledTestPlan, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT p.id, p.account_id, p.model_id, p.cron_expression, p.enabled, p.max_results, p.auto_recover, p.last_run_at, p.next_run_at, p.created_at, p.updated_at, p.pelican_config, p.running_until
 FROM scheduled_test_plans p JOIN accounts a ON a.id=p.account_id
 WHERE p.pelican_config->'quality' IS NOT NULL AND a.deleted_at IS NULL ORDER BY p.id DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanPlans(rows)
}
func (r *scheduledTestPlanRepository) TriggerQuality(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `UPDATE scheduled_test_plans SET next_run_at=NOW(), updated_at=NOW()
 WHERE id=$1 AND enabled AND pelican_config->'quality' IS NOT NULL AND (running_until IS NULL OR running_until<NOW())`, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("plan must be enabled and idle")
	}
	return nil
}

type qualityGroup struct {
	AccountID     int64           `json:"account_id"`
	GroupID       int64           `json:"group_id"`
	Priority      int             `json:"priority"`
	CreatedAt     time.Time       `json:"created_at"`
	AllowedModels json.RawMessage `json:"allowed_models"`
}
type qualityState struct {
	Action         string          `json:"action"`
	AccountVersion time.Time       `json:"account_version"`
	Removed        []qualityGroup  `json:"removed"`
	Remaining      json.RawMessage `json:"remaining"`
}

// Lease/version checks, account mutation, ownership and scheduler invalidation
// commit together. A paused, edited, deleted or expired run cannot change accounts.
func (r *scheduledTestPlanRepository) ApplyQualityOutcome(ctx context.Context, plan *service.ScheduledTestPlan, until time.Time, outcome string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()
	var valid bool
	err = tx.QueryRowContext(ctx, `SELECT enabled AND updated_at=$2 AND running_until=$3 AND running_until>NOW()
 FROM scheduled_test_plans WHERE id=$1 FOR UPDATE`, plan.ID, plan.UpdatedAt, until).Scan(&valid)
	if err == sql.ErrNoRows || (err == nil && !valid) {
		return "stale_run", nil
	}
	if err != nil {
		return "", err
	}
	var version time.Time
	var schedulable bool
	var status string
	var accountName string
	err = tx.QueryRowContext(ctx, `SELECT updated_at, schedulable, status, name FROM accounts WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, plan.AccountID).Scan(&version, &schedulable, &status, &accountName)
	if err == sql.ErrNoRows {
		return "account_deleted", nil
	}
	if err != nil {
		return "", err
	}
	var raw []byte
	err = tx.QueryRowContext(ctx, `SELECT state FROM account_quality_states WHERE plan_id=$1`, plan.ID).Scan(&raw)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	var state qualityState
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &state); err != nil {
			return "", err
		}
	}
	groups, err := qualityGroups(ctx, tx, plan.AccountID)
	if err != nil {
		return "", err
	}
	q := plan.PelicanConfig.Quality
	action := "no_change"
	changed := false
	switch {
	case outcome == "inconclusive":
		action = "inconclusive"
	case outcome == "failed" && state.Action != "":
		action = "already_quarantined"
	case outcome == "failed":
		state.Action = q.Action
		switch q.Action {
		case "disable_scheduling":
			if schedulable {
				_, err = tx.ExecContext(ctx, `UPDATE accounts SET schedulable=false, updated_at=clock_timestamp() WHERE id=$1`, plan.AccountID)
				changed = true
				action = "scheduling_disabled"
			}
		case "remove_groups":
			var all []qualityGroup
			if err = json.Unmarshal(groups, &all); err != nil {
				return "", err
			}
			selected := map[int64]bool{}
			for _, id := range q.RemoveGroupIDs {
				selected[id] = true
			}
			for _, group := range all {
				if selected[group.GroupID] {
					if _, err = tx.ExecContext(ctx, `DELETE FROM account_groups WHERE account_id=$1 AND group_id=$2`, plan.AccountID, group.GroupID); err != nil {
						return "", err
					}
					state.Removed = append(state.Removed, group)
				}
			}
			changed = len(state.Removed) > 0
			if changed {
				action = "groups_removed"
				_, err = tx.ExecContext(ctx, `UPDATE accounts SET updated_at=clock_timestamp() WHERE id=$1`, plan.AccountID)
			}
		}
		if err != nil {
			return "", err
		}
		if changed {
			if err = tx.QueryRowContext(ctx, `SELECT updated_at FROM accounts WHERE id=$1`, plan.AccountID).Scan(&state.AccountVersion); err != nil {
				return "", err
			}
			state.Remaining, err = qualityGroups(ctx, tx, plan.AccountID)
			if err != nil {
				return "", err
			}
			data, marshalErr := json.Marshal(state)
			if marshalErr != nil {
				return "", marshalErr
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO account_quality_states(plan_id,state) VALUES($1,$2)`, plan.ID, string(data)); err != nil {
				return "", err
			}
		}
	case outcome == "passed" && state.Action != "" && q.AutoRestore:
		// Any intervening account/membership edit relinquishes automated restoration.
		if !version.Equal(state.AccountVersion) || !jsonEqual(groups, state.Remaining) || status != "active" {
			return finishQualityOperation(ctx, tx, plan, until, accountName, outcome, "restore_conflict", state)
		}
		switch state.Action {
		case "disable_scheduling":
			_, err = tx.ExecContext(ctx, `UPDATE accounts SET schedulable=true, updated_at=clock_timestamp() WHERE id=$1 AND (expires_at IS NULL OR expires_at>NOW())`, plan.AccountID)
			if err == nil {
				var enabled bool
				err = tx.QueryRowContext(ctx, `SELECT schedulable FROM accounts WHERE id=$1`, plan.AccountID).Scan(&enabled)
				if err == nil && !enabled {
					return finishQualityOperation(ctx, tx, plan, until, accountName, outcome, "restore_conflict", state)
				}
			}
		case "remove_groups":
			for _, group := range state.Removed {
				var exists bool
				if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM groups WHERE id=$1 AND deleted_at IS NULL)`, group.GroupID).Scan(&exists); err != nil {
					return "", err
				}
				if !exists {
					return finishQualityOperation(ctx, tx, plan, until, accountName, outcome, "restore_conflict", state)
				}
			}
			for _, group := range state.Removed {
				var allowed any
				if len(group.AllowedModels) > 0 && string(group.AllowedModels) != "null" {
					allowed = string(group.AllowedModels)
				}
				if _, err = tx.ExecContext(ctx, `INSERT INTO account_groups(account_id,group_id,priority,created_at,allowed_models) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`, plan.AccountID, group.GroupID, group.Priority, group.CreatedAt, allowed); err != nil {
					return "", err
				}
			}
			_, err = tx.ExecContext(ctx, `UPDATE accounts SET updated_at=clock_timestamp() WHERE id=$1`, plan.AccountID)
		}
		if err != nil {
			return "", err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM account_quality_states WHERE plan_id=$1`, plan.ID); err != nil {
			return "", err
		}
		changed = true
		action = "restored"
	case outcome == "passed":
		action = "passed"
	}
	if changed {
		// Include removed groups so their cached scheduling buckets are rebuilt too.
		var all []qualityGroup
		if err = json.Unmarshal(groups, &all); err != nil {
			return "", err
		}
		ids := make([]int64, 0, len(all)+len(state.Removed))
		for _, g := range all {
			ids = append(ids, g.GroupID)
		}
		for _, g := range state.Removed {
			ids = append(ids, g.GroupID)
		}
		payload, marshalErr := json.Marshal(map[string]any{"group_ids": ids})
		if marshalErr != nil {
			return "", marshalErr
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO scheduler_outbox(event_type,account_id,payload) VALUES('account_groups_changed',$1,$2)`, plan.AccountID, string(payload)); err != nil {
			return "", err
		}
	}
	return finishQualityOperation(ctx, tx, plan, until, accountName, outcome, action, state)
}

func finishQualityOperation(ctx context.Context, tx *sql.Tx, plan *service.ScheduledTestPlan, until time.Time, accountName, outcome, action string, state qualityState) (string, error) {
	status := "failed"
	errorMessage := outcome
	if outcome == "passed" {
		status = "success"
		errorMessage = ""
	} else if outcome == "failed" {
		errorMessage = "answer_mismatch"
	}
	var snapshot any
	if state.Action != "" {
		data, err := json.Marshal(state)
		if err != nil {
			return "", err
		}
		snapshot = string(data)
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO account_quality_operations
 (plan_id,account_id,account_name,quality_round_id,outcome,status,error_message,quality_action,pelican_config,snapshot)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
 ON CONFLICT(plan_id,quality_round_id) DO NOTHING`, plan.ID, plan.AccountID, accountName, until.Format(time.RFC3339Nano), service.StableQualityOutcome(outcome, action), status, errorMessage, action, marshalQualityAuditConfig(plan.PelicanConfig), snapshot)
	if err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	return action, nil
}

func qualityGroups(ctx context.Context, tx *sql.Tx, accountID int64) ([]byte, error) {
	var raw []byte
	err := tx.QueryRowContext(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(g) ORDER BY g.group_id),'[]'::jsonb) FROM account_groups g WHERE account_id=$1`, accountID).Scan(&raw)
	return raw, err
}
func jsonEqual(a, b []byte) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	left, _ := json.Marshal(x)
	right, _ := json.Marshal(y)
	return string(left) == string(right)
}

// Global operation history is cursor-paginated independently of account/rule selection.
// Response bodies are loaded through the existing per-result detail endpoint.

func (r *scheduledTestResultRepository) ListQualityHistory(ctx context.Context, beforeID int64, limit int) ([]*service.QualityHistoryResult, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,plan_id,status,error_message,
 GREATEST(0,(EXTRACT(EPOCH FROM (finished_at-started_at))*1000)::bigint),
 started_at,finished_at,created_at,pelican_config,quality_action,quality_judgment,
 quality_round_id,account_id,account_name,passed_count,total_count,result_ids
 FROM account_quality_operations
 WHERE ($1::bigint=0 OR id<$1) ORDER BY id DESC LIMIT $2`, beforeID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]*service.QualityHistoryResult, 0)
	for rows.Next() {
		item := &service.QualityHistoryResult{}
		var cfg, judgment []byte
		if err := rows.Scan(&item.ID, &item.PlanID, &item.Status, &item.ErrorMessage, &item.LatencyMs, &item.StartedAt, &item.FinishedAt, &item.CreatedAt, &cfg, &item.QualityAction, &judgment, &item.QualityRoundID, &item.AccountID, &item.AccountName, &item.PassedCount, &item.TotalCount, pq.Array(&item.ResultIDs)); err != nil {
			return nil, err
		}
		if len(cfg) > 0 {
			if err := json.Unmarshal(cfg, &item.PelicanConfig); err != nil {
				return nil, err
			}
		}
		if len(judgment) > 0 {
			if err := json.Unmarshal(judgment, &item.QualityJudgment); err != nil {
				return nil, err
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// FinalizeQualityOperation attaches retained result IDs and summary data to the
// durable audit row. The upsert also repairs a missing initial row without
// coupling audit retention to scheduled_test_results.
func (r *scheduledTestResultRepository) FinalizeQualityOperation(ctx context.Context, plan *service.ScheduledTestPlan, roundID string, results []*service.ScheduledTestResult) error {
	if plan == nil || plan.PelicanConfig == nil || plan.PelicanConfig.Quality == nil || roundID == "" {
		return nil
	}
	status := "success"
	outcome := "passed"
	errorMessage := ""
	action := ""
	passed := 0
	ids := make([]int64, 0, len(results))
	var started, finished time.Time
	var judgment *service.QualityJudgment
	for _, result := range results {
		if result == nil {
			status, outcome, errorMessage = "failed", "inconclusive", "inconclusive"
			continue
		}
		if started.IsZero() || result.StartedAt.Before(started) {
			started = result.StartedAt
		}
		if result.FinishedAt.After(finished) {
			finished = result.FinishedAt
		}
		if result.ID > 0 {
			ids = append(ids, result.ID)
		}
		if result.QualityAction != "" {
			action = result.QualityAction
		}
		if result.Status == "success" {
			passed++
		} else {
			status, outcome = "failed", "inconclusive"
			if errorMessage == "" {
				errorMessage = result.ErrorMessage
			}
		}
		if result.QualityJudgment != nil {
			if judgment == nil || result.QualityJudgment.Verdict == "incorrect" || (judgment.Verdict == "correct" && result.QualityJudgment.Verdict == "unknown") {
				judgment = result.QualityJudgment
			}
			if result.QualityJudgment.Verdict == "incorrect" && result.ErrorMessage == "answer_mismatch" {
				status, outcome, errorMessage = "failed", "failed", "answer_mismatch"
			}
		}
	}
	if started.IsZero() {
		started = time.Now()
	}
	if finished.IsZero() {
		finished = started
	}
	if action == "" {
		action = "no_change"
	}
	switch action {
	case "stale_run", "account_deleted", "restore_conflict":
		status, outcome, errorMessage = "failed", "skipped", action
	case "action_error":
		status, outcome, errorMessage = "failed", "error", action
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO account_quality_operations
 (plan_id,account_id,account_name,quality_round_id,outcome,status,error_message,quality_action,quality_judgment,pelican_config,passed_count,total_count,result_ids,started_at,finished_at)
 VALUES($1,$2,COALESCE((SELECT name FROM accounts WHERE id=$2),''),$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
 ON CONFLICT(plan_id,quality_round_id) DO UPDATE SET
 outcome=EXCLUDED.outcome,status=EXCLUDED.status,error_message=EXCLUDED.error_message,
 quality_action=EXCLUDED.quality_action,quality_judgment=EXCLUDED.quality_judgment,
 pelican_config=EXCLUDED.pelican_config,passed_count=EXCLUDED.passed_count,
 total_count=EXCLUDED.total_count,result_ids=EXCLUDED.result_ids,
 started_at=EXCLUDED.started_at,finished_at=EXCLUDED.finished_at`,
		plan.ID, plan.AccountID, roundID, service.StableQualityOutcome(outcome, action), status, errorMessage, action,
		marshalQualityJudgment(judgment), marshalQualityAuditConfig(plan.PelicanConfig), passed, len(results), pq.Array(ids), started, finished)
	return err
}

// Operation history is safe to display in the admin dashboard without
// repeating operator-authored probe or grading prompts.
func marshalQualityAuditConfig(config *service.PelicanTestConfig) any {
	if config == nil {
		return nil
	}
	copy := *config
	copy.Prompt = ""
	if config.Quality != nil {
		quality := *config.Quality
		copy.Quality = &quality
		if config.Quality.Judge != nil {
			judge := *config.Quality.Judge
			judge.Prompt = ""
			copy.Quality.Judge = &judge
		}
	}
	data, _ := json.Marshal(&copy)
	return string(data)
}
