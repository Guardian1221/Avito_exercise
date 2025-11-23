.PHONY: build run compose-up migrate

build:
	go build -o bin/prsvc ./cmd/prsvc

run:
	DATABASE_URL=postgres://prusr:prpwd@localhost:5432/prsvc?sslmode=disable ./bin/prsvc

compose-up:
	docker compose up --build

migrate:
	docker run --rm -v $(PWD)/migrations:/migrations --network host migrate/migrate:v4.15.2 -path=/migrations -database "postgres://prusr:prpwd@localhost:5432/prsvc?sslmode=disable" up
