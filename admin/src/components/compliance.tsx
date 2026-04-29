// @ts-nocheck
import React from 'react'
import { ComingSoon } from './coming_soon'

export function CompliancePage() {
  return (
    <ComingSoon
      title="Exporting logs is coming in v0.2"
      message="Audit log export (CSV / JSON), GDPR data subject requests, and SOC2 retention policies ship in v0.2."
      hint="Audit logs are still queryable today via the /audit page and the /api/v1/audit endpoints."
      githubUrl="https://github.com/shark-auth/shark/discussions"
    />
  );
}

export default CompliancePage;
