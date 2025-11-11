# initialize project environment
.PHONY: setup
setup:
	@sh ./scripts/setup-env.sh
	@sh ./scripts/setup-hook.sh

.PHONY fmt
fmt:
	@goimports -l -w $$(find . -type f -name '*.go' -not -path "./.idea/*" -not -path "./.vscode/*")
	@gofumpt -l -w $$(find . -type f -name '*.go' -not -path "./.idea/*" -not -path "./.vscode/*")

# 清理项目依赖
.PHONY tidy
	@go mod tidy -v

.PHONY check
check:
	@$(MAKE) --no-print-directory fmt
	@$(MAKE) --no-print-directory tidy

# code checks
.PHONY lint
lint:
	@golangci-lint run -c ./scripts/lint/.golangci.yaml ./...
