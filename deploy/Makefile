.PHONY: check inventory lint syntax

check:
	./scripts/check-ansible.sh

inventory:
	ansible-inventory --graph

syntax:
	@set -e; for playbook in playbooks/*.yml; do \
		ansible-playbook --syntax-check "$$playbook"; \
	done

lint:
	ansible-lint
