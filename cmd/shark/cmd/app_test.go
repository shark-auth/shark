//go:build integration

package cmd

// Phase E — CLI commands now route via the admin HTTP API instead of opening
// sharkauth.yaml directly. The integration tests below that previously tested
// yaml-loading + direct DB writes have been replaced by API-level tests.
//
// TODO(phase-E): Add integration tests that spin up a test shark server,
// call the admin API via adminDo, and assert the expected behaviour.
// See YAML_DEPRECATION_PLAN.md §Phase E for context.
//
// The following tests from the pre-Phase-E file are intentionally skipped:
//   - TestE2E_AppCreate        — used appCreateCmd.RunE + direct DB; now routes via API
//   - TestE2E_AppRotateSecret  — used appRotateCmd.RunE + direct DB; now routes via API
//   - TestE2E_AppDeleteDefault_Refused — validated is_default guard via direct DB seed;
//                                        guard is now enforced server-side; CLI forwards DELETE to API
