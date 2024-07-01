dev-processor:
	@echo "Running processor"
	@cd processor && air

dev-filer:
	@echo "Running filer"
	@cd filer && air

dev-all:
	@echo "Running all"
	@make dev-processor &
	@make dev-filer &
