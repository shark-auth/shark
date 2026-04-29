// @ts-nocheck
import React from 'react'
import { ComingSoon } from './coming_soon'

export function Proxy() {
  return (
    <ComingSoon
      title="Proxy is coming in v0.2"
      message="The SharkAuth reverse proxy gateway â€” per-route auth, token injection, and upstream config â€” is under active development and will ship in v0.2."
      hint="Self-hosted users can preview by setting VITE_FEATURE_PROXY=true"
      githubUrl="https://github.com/shark-auth/shark/discussions"
    />
  );
}

export default Proxy;
