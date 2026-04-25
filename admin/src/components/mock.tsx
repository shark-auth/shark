// @ts-nocheck

const NOW = Date.now();
const mins = (n) => NOW - n * 60 * 1000;
const hrs = (n) => NOW - n * 60 * 60 * 1000;
const days = (n) => NOW - n * 24 * 60 * 60 * 1000;

export const MOCK = {
  stats: {
    users: { total: 12847, delta7d: 312 },
    sessions: { active: 3921 },
    mfa: { pct: 0.68, enabled: 8735, total: 12847 },
    failedLogins24h: { count: 142, deltaPct: -0.18 },
    apiKeys: { count: 47, expiring: 3 },
    agents: { count: 23, tokensLive: 1284 },
  },
  trends: {
    users: [120, 125, 128, 132, 135, 138, 142, 145, 149, 152, 156, 161, 164, 168],
    sessions: [340, 380, 362, 410, 425, 398, 421, 446, 412, 438, 462, 481, 465, 492],
    mfa: [61, 62, 62, 63, 63, 64, 64, 65, 65, 66, 67, 67, 68, 68],
    failed: [180, 165, 172, 158, 190, 175, 160, 155, 168, 152, 145, 140, 148, 142],
    keys: [42, 42, 43, 43, 44, 44, 45, 45, 46, 46, 47, 47, 47, 47],
    agents: [420, 512, 640, 812, 905, 1020, 1115, 1180, 1195, 1212, 1240, 1255, 1270, 1284],
  },
  authBreakdown: [
    { label: 'Password', value: 4210, color: '#e4e4e4' },
    { label: 'OAuth', value: 3820, color: '#888' },
    { label: 'Passkey', value: 2140, color: '#555' },
    { label: 'Magic link', value: 1680, color: '#3a3a3a' },
    { label: 'SSO', value: 997, color: '#262626' },
  ],
  attention: [
    { id: 1, kind: 'device', severity: 'warn', title: '4 pending device approvals', sub: 'Oldest requested 2m ago \u00b7 claude-cli, cursor-agent', action: 'Review', href: 'device-flow' },
    { id: 2, kind: 'webhook', severity: 'danger', title: '12 webhook deliveries failing', sub: 'endpoint events-prod \u2192 503 (5 consecutive)', action: 'Investigate', href: 'overview' },
    { id: 3, kind: 'key', severity: 'warn', title: '3 API keys expiring in 7 days', sub: 'sk_live_\u2026aK9f, sk_live_\u2026m2Bc, ci-bot', action: 'Rotate', href: 'overview' },
    { id: 4, kind: 'config', severity: 'info', title: 'shark.email sandbox mode in use', sub: 'Switch provider before production \u2014 280/500 daily quota', action: 'Set up', href: 'overview' },
    { id: 5, kind: 'anomaly', severity: 'danger', title: 'Brute force pattern detected', sub: '47 failed logins from 185.220.101.\u00b7 in 4m \u00b7 blocked', action: 'Triage', href: 'overview' },
  ],
  activity: [
    { t: mins(0.2), actor: 'agent', name: 'cursor-agent', action: 'oauth.token.issued', target: 'usr_7fK2\u2026', meta: 'scope: profile email' },
    { t: mins(0.5), actor: 'user', name: 'amelia@nimbus.sh', action: 'user.login', target: '\u2014', meta: 'passkey \u00b7 iOS' },
    { t: mins(1), actor: 'system', name: 'shark', action: 'webhook.delivery.failed', target: 'wh_a83\u2026', meta: '503 \u2014 retry in 30s' },
    { t: mins(2), actor: 'admin', name: 'you', action: 'agent.secret.rotated', target: 'agent_cursor', meta: '\u2014' },
    { t: mins(3), actor: 'agent', name: 'claude-cli', action: 'device.flow.started', target: '\u2014', meta: 'code: KXQP-7MNT' },
    { t: mins(5), actor: 'user', name: 'dev@hexcel.co', action: 'mfa.enabled', target: '\u2014', meta: 'totp' },
    { t: mins(8), actor: 'agent', name: 'lindy-ops', action: 'oauth.token.issued', target: 'usr_9mZ4\u2026', meta: 'act: svc-worker' },
    { t: mins(11), actor: 'user', name: 'priya.nair@stride.io', action: 'user.created', target: '\u2014', meta: 'oauth.google' },
    { t: mins(14), actor: 'system', name: 'shark', action: 'signing_key.rotation', target: 'kid: 2026-04-b', meta: 'auto-rotation' },
    { t: mins(17), actor: 'agent', name: 'gpt-5-code', action: 'token.introspected', target: 'jti: 8f\u20262a', meta: 'active=true' },
    { t: mins(20), actor: 'user', name: 'tomas@orbit.so', action: 'user.login', target: '\u2014', meta: 'password \u00b7 macOS' },
    { t: mins(24), actor: 'admin', name: 'you', action: 'app.redirect_uri.added', target: 'app_billing', meta: '+3 uris' },
  ],
  agents: [
    { id: 'agent_cursor', name: 'Cursor', type: 'confidential', clientId: 'cli_01HNGZK8RXT4A2VPQW', status: 'active', grants: ['authorization_code', 'refresh_token'], scopes: ['openid','profile','email','repos:read'], tokensActive: 482, lastUsed: mins(0.2), createdBy: 'amelia@nimbus.sh', dpop: true, description: 'AI code editor \u2014 user-delegated access' },
    { id: 'agent_claude_cli', name: 'Claude CLI', type: 'public', clientId: 'cli_01HNGZKA4BNFEJQP7M', status: 'active', grants: ['device_code', 'refresh_token'], scopes: ['openid','profile','email','workspace:read','workspace:write'], tokensActive: 213, lastUsed: mins(3), createdBy: 'amelia@nimbus.sh', dpop: true, description: 'Terminal AI assistant \u2014 device flow' },
    { id: 'agent_lindy', name: 'Lindy Ops', type: 'confidential', clientId: 'cli_01HNGZKC9FWXTMA8LZ', status: 'active', grants: ['client_credentials', 'token_exchange'], scopes: ['agents:act','tickets:rw','reports:read'], tokensActive: 318, lastUsed: mins(8), createdBy: 'sasha@apex.dev', dpop: true, description: 'Autonomous ops agent \u2014 service account' },
    { id: 'agent_gpt5', name: 'GPT-5 Code', type: 'confidential', clientId: 'cli_01HNGZKDPM2QR5VTYN', status: 'active', grants: ['authorization_code'], scopes: ['openid','profile','email','files:read'], tokensActive: 167, lastUsed: mins(17), createdBy: 'priya.nair@stride.io', dpop: false, description: 'OpenAI dev agent' },
    { id: 'agent_zapier_mcp', name: 'Zapier MCP', type: 'confidential', clientId: 'cli_01HNGZKEV7XY3HJ2KN', status: 'active', grants: ['authorization_code','token_exchange'], scopes: ['workspace:read','webhooks:rw'], tokensActive: 74, lastUsed: hrs(1), createdBy: 'amelia@nimbus.sh', dpop: true, description: 'Workflow automation bridge' },
    { id: 'agent_slack_ai', name: 'Slack AI Bridge', type: 'confidential', clientId: 'cli_01HNGZKG8TJAM5EBRW', status: 'active', grants: ['authorization_code','refresh_token'], scopes: ['openid','profile','messages:read'], tokensActive: 21, lastUsed: hrs(2), createdBy: 'tomas@orbit.so', dpop: false, description: 'Workspace notifications' },
    { id: 'agent_replit', name: 'Replit Agent', type: 'public', clientId: 'cli_01HNGZKHD3QZP8KVY4', status: 'active', grants: ['device_code'], scopes: ['openid','profile','projects:rw'], tokensActive: 9, lastUsed: hrs(3), createdBy: 'kenji@hexcel.co', dpop: true, description: 'In-browser IDE agent' },
    { id: 'agent_legacy_ci', name: 'Legacy CI bot', type: 'confidential', clientId: 'cli_01HNGZKJMRX6FAW2QP', status: 'disabled', grants: ['client_credentials'], scopes: ['deploys:write'], tokensActive: 0, lastUsed: days(28), createdBy: 'soren@hexcel.co', dpop: false, description: 'Deprecated \u2014 superseded by GitHub Actions' },
  ],
  devicePending: [
    { id: 'dev_01', userCode: 'KXQP-7MNT', agent: 'agent_claude_cli', agentName: 'Claude CLI', scopes: ['openid','profile','workspace:read','workspace:write'], resource: 'https://api.nimbus.sh', ip: '73.162.44.11', location: 'San Francisco, CA', device: 'macOS \u00b7 iTerm2', requestedAt: mins(2.1), expiresAt: mins(-7.9), dpop: true, user: 'amelia@nimbus.sh' },
    { id: 'dev_02', userCode: 'BRXN-9JQD', agent: 'agent_cursor', agentName: 'Cursor', scopes: ['openid','profile','email','repos:read'], resource: 'https://api.nimbus.sh', ip: '98.97.132.4', location: 'Austin, TX', device: 'macOS \u00b7 Cursor 0.42', requestedAt: mins(0.8), expiresAt: mins(-9.2), dpop: true, user: 'priya.nair@stride.io' },
    { id: 'dev_03', userCode: 'HMFT-2KLV', agent: 'agent_replit', agentName: 'Replit Agent', scopes: ['openid','profile','projects:rw'], resource: 'https://api.nimbus.sh', ip: '45.63.122.80', location: 'Dublin, IE', device: 'Chrome 124 \u00b7 Linux', requestedAt: mins(4.5), expiresAt: mins(-5.5), dpop: true, user: 'kenji@hexcel.co' },
    { id: 'dev_04', userCode: 'ZQVW-6BPX', agent: 'agent_gpt5', agentName: 'GPT-5 Code', scopes: ['openid','profile','email','files:read'], resource: 'https://api.nimbus.sh', ip: '104.28.82.19', location: 'Berlin, DE', device: 'VSCode \u00b7 1.94', requestedAt: mins(6.2), expiresAt: mins(-3.8), dpop: false, user: 'dev@hexcel.co' },
  ],
  deviceRecent: [
    { userCode: 'MKPR-3FGH', agent: 'Cursor', user: 'tomas@orbit.so', outcome: 'approved', when: mins(12) },
    { userCode: 'TWXJ-8LNM', agent: 'Claude CLI', user: 'amelia@nimbus.sh', outcome: 'approved', when: mins(23) },
    { userCode: 'ADFV-5QPR', agent: 'Lindy Ops', user: 'sasha@apex.dev', outcome: 'expired', when: mins(41) },
    { userCode: 'YCBN-2XWE', agent: 'Cursor', user: 'priya.nair@stride.io', outcome: 'approved', when: mins(58) },
    { userCode: 'JHGF-7RTY', agent: 'Slack AI Bridge', user: 'tomas@orbit.so', outcome: 'denied', when: hrs(2) },
    { userCode: 'QAZX-4CDE', agent: 'Claude CLI', user: 'amelia@nimbus.sh', outcome: 'approved', when: hrs(3) },
    { userCode: 'WERT-9POI', agent: 'Cursor', user: 'zara@apex.dev', outcome: 'approved', when: hrs(4) },
  ],
  relativeTime(t) {
    const diff = Math.floor((NOW - t) / 1000);
    if (diff < 0) {
      const n = Math.abs(diff);
      if (n < 60) return `in ${n}s`;
      if (n < 3600) return `in ${Math.floor(n/60)}m`;
      if (n < 86400) return `in ${Math.floor(n/3600)}h`;
      return `in ${Math.floor(n/86400)}d`;
    }
    if (diff < 60) return `${diff}s ago`;
    if (diff < 3600) return `${Math.floor(diff/60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff/3600)}h ago`;
    return `${Math.floor(diff/86400)}d ago`;
  },
};
