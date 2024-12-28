SUBDIR_CLIENT := client
SUBDIR_CONTROL := control
SUBDIR_API := api
DEFAULT_ETCD_DIR := ./default.etcd
MODULES := ./client ./control ./data-generator
MODULE_ROOT_NAME := csb

.PHONY: gen client control run-client run-control clean test

all:
	@$(MAKE) --no-print-directory -C $(SUBDIR_CONTROL) all
	@$(MAKE) --no-print-directory -C $(SUBDIR_CLIENT) all

client:
	@$(MAKE) --no-print-directory -C $(SUBDIR_CLIENT) all

control:
	@$(MAKE) --no-print-directory -C $(SUBDIR_CONTROL) all

gen:
	@$(MAKE) --no-print-directory -C $(SUBDIR_API) gen

clean:
	rm -rf $(DEFAULT_ETCD_DIR)
	@$(MAKE) --no-print-directory -C $(SUBDIR_CONTROL) clean
	@$(MAKE) --no-print-directory -C $(SUBDIR_CLIENT) clean

test:
	@for dir in $(MODULES); do \
		echo "Testing $$dir..."; \
		(cd $$dir && go clean -testcache && go test -v ./...) || exit 1; \
	done
