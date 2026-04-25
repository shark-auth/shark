.PHONY: install seed run revoke audit clean

PYTHON ?= python3

install:
	$(PYTHON) -m pip install -e ../../sdk/python
	$(PYTHON) -m pip install requests pyjwt cryptography python-dotenv

seed:
	bash seed.sh

run:
	$(PYTHON) orchestrator/main.py

revoke-email:
	$(PYTHON) orchestrator/revoke.py email-agent --reason key_compromise

audit:
	$(PYTHON) audit_replay.py --last-run

mock-kb:
	$(PYTHON) resources/mock_resource.py --port 9101 --audience kb-api --required-scope kb:read

mock-smtp:
	$(PYTHON) resources/mock_resource.py --port 9102 --audience smtp-relay --required-scope email:send

mock-crm:
	$(PYTHON) resources/mock_resource.py --port 9103 --audience salesforce-api --required-scope crm:write

clean:
	rm -f .env .last_run.json
