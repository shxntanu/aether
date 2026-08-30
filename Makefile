.PHONY: backend-fmt backend-run backend-test backend-vet dev frontend-build frontend-dev frontend-lint frontend-test test

backend-fmt:
	cd backend && gofmt -w $$(find . -name '*.go' -type f)

backend-run:
	cd backend && go run ./cmd/aether

backend-test:
	cd backend && go test ./...

backend-vet:
	cd backend && go vet ./...

frontend-dev:
	cd frontend && pnpm dev

frontend-build:
	cd frontend && pnpm build

frontend-lint:
	cd frontend && pnpm lint

frontend-test:
	cd frontend && pnpm test

test: backend-test frontend-test

dev:
	@echo "Run 'make backend-run' and 'make frontend-dev' in separate terminals."
