upstream_ref ?= 9cd304f78a5c12401c9711cbd56d2a12eaca0632

.PHONY: fetch-nightscout-tests
fetch-nightscout-tests:
	./scripts/fetch_nightscout_tests.sh $(upstream_ref)
