.PHONY: run generate test tidy clean seed

run: generate seed
	go run ./cmd/server

generate:
	templ generate

test:
	go test ./...

tidy:
	go mod tidy

seed:
	go run ./cmd/seed

clean:
	rm -f recipes.db recipes.db-wal recipes.db-shm
