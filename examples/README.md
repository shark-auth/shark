# SharkAuth examples

Runnable end-to-end demos. Every script in this directory is exercised
by the Hello Agent walkthrough or by the smoke test suite.

| file | what it does |
|------|-------------|
| [`hello_agent.sh`](hello_agent.sh) | Build shark, start dev server, DCR-register an agent, run `hello_agent.py`, clean up. See [`docs/hello-agent.md`](../docs/hello-agent.md). |
| [`hello_agent.py`](hello_agent.py) | Mint a `client_credentials` token, build a DPoP proof, verify claims via RFC 7662 introspection. Depends on `shark_auth` + `requests`. |
