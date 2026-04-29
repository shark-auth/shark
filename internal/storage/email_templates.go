package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ListEmailTemplates returns all email templates ordered by id.
func (s *SQLiteStore) ListEmailTemplates(ctx context.Context) ([]*EmailTemplate, error) {
	rows, err := s.reader.QueryContext(ctx, `
		SELECT id, subject, preheader, header_text, body_paragraphs, cta_text, cta_url_template, footer_text, updated_at
		FROM email_templates ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*EmailTemplate{}
	for rows.Next() {
		t := &EmailTemplate{}
		var paragraphsJSON string
		err := rows.Scan(&t.ID, &t.Subject, &t.Preheader, &t.HeaderText,
			&paragraphsJSON, &t.CTAText, &t.CTAURLTemplate, &t.FooterText, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(paragraphsJSON), &t.BodyParagraphs)
		if t.BodyParagraphs == nil {
			t.BodyParagraphs = []string{}
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetEmailTemplate returns one template by id.
func (s *SQLiteStore) GetEmailTemplate(ctx context.Context, id string) (*EmailTemplate, error) {
	row := s.reader.QueryRowContext(ctx, `
		SELECT id, subject, preheader, header_text, body_paragraphs, cta_text, cta_url_template, footer_text, updated_at
		FROM email_templates WHERE id = ?`, id)
	t := &EmailTemplate{}
	var paragraphsJSON string
	err := row.Scan(&t.ID, &t.Subject, &t.Preheader, &t.HeaderText,
		&paragraphsJSON, &t.CTAText, &t.CTAURLTemplate, &t.FooterText, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("email template not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(paragraphsJSON), &t.BodyParagraphs)
	if t.BodyParagraphs == nil {
		t.BodyParagraphs = []string{}
	}
	return t, nil
}

// UpdateEmailTemplate applies a partial update. Unknown fields are dropped.
// body_paragraphs may be []string or []any — both get JSON-marshalled.
func (s *SQLiteStore) UpdateEmailTemplate(ctx context.Context, id string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	allowed := map[string]bool{
		"subject": true, "preheader": true, "header_text": true,
		"body_paragraphs": true, "cta_text": true, "cta_url_template": true, "footer_text": true,
	}
	setParts := []string{}
	args := []any{}
	for k, v := range fields {
		if !allowed[k] {
			continue
		}
		if k == "body_paragraphs" {
			if arr, ok := v.([]string); ok {
				b, _ := json.Marshal(arr)
				v = string(b)
			} else if arr, ok := v.([]any); ok {
				b, _ := json.Marshal(arr)
				v = string(b)
			}
		}
		setParts = append(setParts, k+" = ?")
		args = append(args, v)
	}
	if len(setParts) == 0 {
		return nil
	}
	setParts = append(setParts, "updated_at = ?")
	args = append(args, time.Now().UTC().Format(time.RFC3339))
	args = append(args, id)
	q := "UPDATE email_templates SET " + strings.Join(setParts, ", ") + " WHERE id = ?"
	_, err := s.writer.ExecContext(ctx, q, args...)
	return err
}

// SeedEmailTemplates inserts V1 templates if missing. Idempotent via INSERT OR IGNORE.
func (s *SQLiteStore) SeedEmailTemplates(ctx context.Context) error {
	seeds := defaultEmailTemplateSeeds()
	now := time.Now().UTC().Format(time.RFC3339)
	for _, seed := range seeds {
		pJSON, _ := json.Marshal(seed.BodyParagraphs)
		_, err := s.writer.ExecContext(ctx, `
			INSERT OR IGNORE INTO email_templates
			(id, subject, preheader, header_text, body_paragraphs, cta_text, cta_url_template, footer_text, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			seed.ID, seed.Subject, seed.Preheader, seed.HeaderText, string(pJSON),
			seed.CTAText, seed.CTAURLTemplate, seed.FooterText, now)
		if err != nil {
			return fmt.Errorf("seed %s: %w", seed.ID, err)
		}
	}
	return nil
}
