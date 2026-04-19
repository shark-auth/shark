package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// --- Auth Flows ---

func (s *SQLiteStore) CreateAuthFlow(ctx context.Context, flow *AuthFlow) error {
	stepsJSON, err := marshalFlowSteps(flow.Steps)
	if err != nil {
		return fmt.Errorf("marshal steps: %w", err)
	}
	condsJSON, err := marshalMetadata(flow.Conditions)
	if err != nil {
		return fmt.Errorf("marshal conditions: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO auth_flows
		 (id, name, trigger, steps, enabled, priority, conditions, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		flow.ID, flow.Name, flow.Trigger, stepsJSON,
		boolToInt(flow.Enabled), flow.Priority, condsJSON,
		flow.CreatedAt.UTC().Format(time.RFC3339),
		flow.UpdatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteStore) GetAuthFlowByID(ctx context.Context, id string) (*AuthFlow, error) {
	return s.scanAuthFlow(s.db.QueryRowContext(ctx,
		`SELECT id, name, trigger, steps, enabled, priority, conditions, created_at, updated_at
		 FROM auth_flows WHERE id = ?`, id))
}

func (s *SQLiteStore) ListAuthFlows(ctx context.Context) ([]*AuthFlow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, trigger, steps, enabled, priority, conditions, created_at, updated_at
		 FROM auth_flows ORDER BY priority DESC, created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*AuthFlow
	for rows.Next() {
		f, err := s.scanAuthFlowFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ListAuthFlowsByTrigger returns flows for a given trigger, highest priority
// first. Ties are broken by created_at ASC (earliest wins) so behavior is
// deterministic under concurrent inserts.
func (s *SQLiteStore) ListAuthFlowsByTrigger(ctx context.Context, trigger string) ([]*AuthFlow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, trigger, steps, enabled, priority, conditions, created_at, updated_at
		 FROM auth_flows WHERE trigger = ?
		 ORDER BY priority DESC, created_at ASC`, trigger)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*AuthFlow
	for rows.Next() {
		f, err := s.scanAuthFlowFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) UpdateAuthFlow(ctx context.Context, flow *AuthFlow) error {
	stepsJSON, err := marshalFlowSteps(flow.Steps)
	if err != nil {
		return fmt.Errorf("marshal steps: %w", err)
	}
	condsJSON, err := marshalMetadata(flow.Conditions)
	if err != nil {
		return fmt.Errorf("marshal conditions: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE auth_flows SET
		   name = ?, trigger = ?, steps = ?, enabled = ?,
		   priority = ?, conditions = ?, updated_at = ?
		 WHERE id = ?`,
		flow.Name, flow.Trigger, stepsJSON,
		boolToInt(flow.Enabled), flow.Priority, condsJSON,
		time.Now().UTC().Format(time.RFC3339),
		flow.ID,
	)
	return err
}

func (s *SQLiteStore) DeleteAuthFlow(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM auth_flows WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) scanAuthFlow(row *sql.Row) (*AuthFlow, error) {
	var f AuthFlow
	var stepsJSON, condsJSON string
	var enabled int
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&f.ID, &f.Name, &f.Trigger, &stepsJSON,
		&enabled, &f.Priority, &condsJSON,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}
	return unmarshalAuthFlow(&f, stepsJSON, condsJSON, enabled, createdAtStr, updatedAtStr)
}

func (s *SQLiteStore) scanAuthFlowFromRows(rows *sql.Rows) (*AuthFlow, error) {
	var f AuthFlow
	var stepsJSON, condsJSON string
	var enabled int
	var createdAtStr, updatedAtStr string

	err := rows.Scan(
		&f.ID, &f.Name, &f.Trigger, &stepsJSON,
		&enabled, &f.Priority, &condsJSON,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}
	return unmarshalAuthFlow(&f, stepsJSON, condsJSON, enabled, createdAtStr, updatedAtStr)
}

func unmarshalAuthFlow(
	f *AuthFlow,
	stepsJSON, condsJSON string,
	enabled int,
	createdAtStr, updatedAtStr string,
) (*AuthFlow, error) {
	f.Enabled = enabled != 0

	steps, err := unmarshalFlowSteps(stepsJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshal steps: %w", err)
	}
	f.Steps = steps

	if err := json.Unmarshal([]byte(condsJSON), &f.Conditions); err != nil {
		return nil, fmt.Errorf("unmarshal conditions: %w", err)
	}
	if f.Conditions == nil {
		f.Conditions = map[string]any{}
	}

	f.CreatedAt, err = parseAuthFlowTime(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at %q: %w", createdAtStr, err)
	}
	f.UpdatedAt, err = parseAuthFlowTime(updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at %q: %w", updatedAtStr, err)
	}
	return f, nil
}

// --- Auth Flow Runs ---

func (s *SQLiteStore) CreateAuthFlowRun(ctx context.Context, run *AuthFlowRun) error {
	metaJSON, err := marshalMetadata(run.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	var userID interface{}
	if run.UserID != "" {
		userID = run.UserID
	}
	var reason interface{}
	if run.Reason != "" {
		reason = run.Reason
	}
	var blockedAt interface{}
	if run.BlockedAtStep != nil {
		blockedAt = int64(*run.BlockedAtStep)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO auth_flow_runs
		 (id, flow_id, user_id, trigger, outcome, blocked_at_step,
		  reason, metadata, started_at, finished_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.FlowID, userID, run.Trigger, run.Outcome,
		blockedAt, reason, metaJSON,
		run.StartedAt.UTC().Format(time.RFC3339),
		run.FinishedAt.UTC().Format(time.RFC3339),
	)
	return err
}

// ListAuthFlowRunsByFlowID returns newest-first runs for a given flow.
// A non-positive limit falls back to 50; values above 500 are clamped.
func (s *SQLiteStore) ListAuthFlowRunsByFlowID(ctx context.Context, flowID string, limit int) ([]*AuthFlowRun, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, flow_id, user_id, trigger, outcome, blocked_at_step,
		        reason, metadata, started_at, finished_at
		 FROM auth_flow_runs
		 WHERE flow_id = ?
		 ORDER BY started_at DESC, id DESC
		 LIMIT ?`, flowID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*AuthFlowRun
	for rows.Next() {
		r, err := s.scanAuthFlowRunFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) scanAuthFlowRunFromRows(rows *sql.Rows) (*AuthFlowRun, error) {
	var r AuthFlowRun
	var userID, reason sql.NullString
	var blockedAt sql.NullInt64
	var metaJSON string
	var startedAtStr, finishedAtStr string

	err := rows.Scan(
		&r.ID, &r.FlowID, &userID, &r.Trigger, &r.Outcome,
		&blockedAt, &reason, &metaJSON,
		&startedAtStr, &finishedAtStr,
	)
	if err != nil {
		return nil, err
	}
	return unmarshalAuthFlowRun(&r, userID, reason, blockedAt, metaJSON, startedAtStr, finishedAtStr)
}

func unmarshalAuthFlowRun(
	r *AuthFlowRun,
	userID, reason sql.NullString,
	blockedAt sql.NullInt64,
	metaJSON string,
	startedAtStr, finishedAtStr string,
) (*AuthFlowRun, error) {
	if userID.Valid {
		r.UserID = userID.String
	}
	if reason.Valid {
		r.Reason = reason.String
	}
	if blockedAt.Valid {
		v := int(blockedAt.Int64)
		r.BlockedAtStep = &v
	}

	if err := json.Unmarshal([]byte(metaJSON), &r.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	if r.Metadata == nil {
		r.Metadata = map[string]any{}
	}

	var err error
	r.StartedAt, err = parseAuthFlowTime(startedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse started_at %q: %w", startedAtStr, err)
	}
	r.FinishedAt, err = parseAuthFlowTime(finishedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse finished_at %q: %w", finishedAtStr, err)
	}
	return r, nil
}

// --- Helpers ---

// marshalFlowSteps JSON-encodes a []FlowStep, always returning "[]" for
// nil/empty so the NOT NULL DEFAULT '[]' invariant on the steps column holds
// even when callers forget to allocate a slice.
func marshalFlowSteps(steps []FlowStep) (string, error) {
	if len(steps) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(steps)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// unmarshalFlowSteps decodes the JSON column into a []FlowStep, normalising
// a missing/empty column to a non-nil empty slice so callers never deal with
// nil on the read path.
func unmarshalFlowSteps(s string) ([]FlowStep, error) {
	if s == "" {
		return []FlowStep{}, nil
	}
	var out []FlowStep
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []FlowStep{}
	}
	return out, nil
}

// parseAuthFlowTime accepts both RFC3339 (what we write) and SQLite's default
// CURRENT_TIMESTAMP format ("2006-01-02 15:04:05", no T separator). Mirrors
// parseVaultTime so the two layers share dual-format tolerance.
func parseAuthFlowTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02 15:04:05", s)
}
