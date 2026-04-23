# Smoke Test Mapping (Bash to Python)

This file tracks the migration progress of `smoke_test.sh` sections to the `pytest` suite.

| Bash Section | Python Test Function | File | Status |
|--------------|----------------------|------|--------|
| 1 & 2 | `test_signup_login_flow` | `test_auth.py` | ✅ |
| 3 | `test_dual_accept_middleware` | `test_auth.py` | ⚠️ Partially Implemented |
| 4 | `test_jwks_parity` | `test_metadata.py` | ✅ |
| 5 | `test_session_list_and_revocation` | `test_user_sessions.py` | ✅ |
| 6 | *Pending* | `test_admin_mgmt.py` | ❌ |
| 7 | `test_key_rotation_cli` | `test_cli_ops.py` | ✅ |
| 8 | `test_apps_cli` | `test_cli_ops.py` | ✅ |
| 9 | `test_apps_http_crud` | `test_admin_mgmt.py` | ✅ |
| 10 | *Pending* | `test_auth.py` | ❌ |
| 11 | `test_organization_rbac` | `test_org_rbac.py` | ✅ |
| 12 | `test_audit_logs_exist` | `test_stats_audit.py` | ✅ |
| 13 | `test_admin_stats` | `test_stats_audit.py` | ✅ |
| 14 | `test_admin_system_endpoints` | `test_cli_ops.py` | ✅ |
| 15 | *Pending* | `test_admin_mgmt.py` | ❌ |
| 16 & 17 | `test_session_list_and_revocation` | `test_user_sessions.py` | ✅ |
| 18 | *Pending* | `test_stats_audit.py` | ❌ |
| 19 | `test_webhooks_crud` | `test_admin_mgmt.py` | ✅ |
| 20 | `test_api_key_crud` | `test_admin_mgmt.py` | ✅ |
| 21 | *Pending* | `test_admin_mgmt.py` | ❌ |
| 22 | *Pending* | `test_admin_mgmt.py` | ❌ |
| 23 | `test_password_change` | `test_user_sessions.py` | ✅ |
| 24 | `test_sso_connections_crud` | `test_admin_mgmt.py` | ✅ |
| 25 | *Pending* | `test_admin_mgmt.py` | ❌ |
| 26 & 28 | `test_as_metadata_rfc8414` | `test_metadata.py` | ✅ |
| 27 | *Implicit in DB checks* | `test_stats_audit.py` | ✅ |
| 29 | `test_agent_crud` | `test_oauth_flows.py` | ✅ |
| 30 | `test_client_credentials_grant` | `test_oauth_flows.py` | ✅ |
| 31 | `test_auth_code_pkce_flow` | `test_oauth_flows.py` | ✅ |
| 32 | `test_pkce_enforcement` | `test_oauth_advanced.py` | ✅ |
| 33 | `test_refresh_token_rotation_and_reuse` | `test_oauth_advanced.py` | ✅ |
| 34 | `test_device_flow` | `test_oauth_advanced.py` | ✅ |
| 35 | `test_token_exchange` | `test_oauth_advanced.py` | ✅ |
| 36 | `test_dpop_surface` | `test_oauth_advanced.py` | ✅ |
| 37 & 38 | `test_introspection_revocation` | `test_oauth_advanced.py` | ✅ |
| 39 | `test_dcr_lifecycle` | `test_oauth_advanced.py` | ✅ |
| 40 | `test_resource_indicators` | `test_oauth_advanced.py` | ✅ |
| 41 | `test_jwks_es256` | `test_oauth_advanced.py` | ✅ |
| 42 | `test_consent_management` | `test_vault_proxy_flows.py` | ✅ |
| 43 & 44 | `test_vault_and_templates` | `test_vault_proxy_flows.py` | ✅ |
| 45 | `test_vault_connect_flow` | `test_vault_proxy_flows.py` | ✅ |
| 50-54 | `test_auth_flow_lifecycle` | `test_vault_proxy_flows.py` | ✅ |
| 52 (Re-test) | `test_flow_blocks_signup` | `test_vault_proxy_flows.py` | ✅ |
| 55 | `test_webhook_replay` | `test_admin_deep.py` | ✅ |
| 56 & 57 | `test_admin_org_mgmt` | `test_admin_deep.py` | ✅ |
| 58 | `test_admin_mfa_disable` | `test_admin_deep.py` | ✅ |
| 59 | `test_audit_complex_filters` | `test_admin_deep.py` | ✅ |
| 60 | `test_failed_logins_stats` | `test_admin_deep.py` | ✅ |
| 66 | `test_rbac_compliance` | `test_rbac_compliance.py` | ✅ |
| 67 & 68 | `test_proxy_rules_crud` | `test_rbac_compliance.py` | ✅ |
| 69 | `test_audit_csv_export` | `test_rbac_compliance.py` | ✅ |
| 70 | `test_admin_user_creation` | `test_rbac_compliance.py` | ✅ |
| 70 (Second) | `test_transparent_gateway_porter` | `test_w15_gateway.py` | ✅ |
| 71 | `test_bootstrap_token_surface` | `test_w15_gateway.py` | ✅ |
| 74 | `test_branding_crud` | `test_w15_gateway.py` | ✅ |
| 75 | `test_hosted_pages_shell` | `test_w15_gateway.py` | ✅ |
| 72 & 73 | *Pending* | `test_w15_advanced.py` | ❌ |
| 76 | *Pending* | `test_sdk_integration.py` | ❌ |
| F4 & F5 | *Partial in Section 35/36* | `test_oauth_advanced.py` | ✅ |
