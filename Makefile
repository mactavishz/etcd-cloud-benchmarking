SUBDIR_CLIENT=client
SUBDIR_CONTROL=control
DEFAULT_ETCD_DIR=./default.etcd
MODULES := ./client ./control ./data-generator ./trace-generator

.PHONY: client control run-client run-control clean test

all:
	@$(MAKE) --no-print-directory -C $(SUBDIR_CONTROL) all
	@$(MAKE) --no-print-directory -C $(SUBDIR_CLIENT) all

client:
	@$(MAKE) --no-print-directory -C $(SUBDIR_CLIENT) all

control:
	@$(MAKE) --no-print-directory -C $(SUBDIR_CONTROL) all

run-client:
	@$(MAKE) --no-print-directory -C $(SUBDIR_CLIENT) run

run-control:
	@$(MAKE) --no-print-directory -C $(SUBDIR_CONTROL) run

clean:
	rm -rf $(DEFAULT_ETCD_DIR)
	@$(MAKE) --no-print-directory -C $(SUBDIR_CONTROL) clean
	@$(MAKE) --no-print-directory -C $(SUBDIR_CLIENT) clean

test:
	@for dir in $(MODULES); do \
		echo "Testing $$dir..."; \
		(cd $$dir && go test -v ./...) || exit 1; \
	done
