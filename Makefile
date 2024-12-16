SUBDIR_CLIENT=client
SUBDIR_CONTROL=control

.PHONY: client control

client:
	@$(MAKE) --no-print-directory -C $(SUBDIR_CLIENT) all

control:
	@$(MAKE) --no-print-directory -C $(SUBDIR_CONTROL) all

run-client:
	@$(MAKE) --no-print-directory -C $(SUBDIR_CLIENT) run

run-control:
	@$(MAKE) --no-print-directory -C $(SUBDIR_CONTROL) run
