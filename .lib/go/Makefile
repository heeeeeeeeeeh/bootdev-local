.PHONY: */* */*/*

*/*/*:
	@echo "Building $@..."
	@if [ ! -f "$@/go.mod" ]; then \
		cd $@ && go mod init $@ && go mod tidy; \
	fi
	cd $@ &&	go test

*/*:
	@echo "Building $@..."
	@if [ ! -f "$@/go.mod" ]; then \
		cd $@ && go mod init $@ && go mod tidy; \
	fi
	cd $@ &&	go test
