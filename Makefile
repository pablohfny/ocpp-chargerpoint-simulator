.PHONY: build run run-local run-dev clean

# Local overrides (copy .env.example to .env); never committed
-include .env
export

# Build the simulator
build:
	CGO_ENABLED=0 go build -o simulator .

# Run with arguments (usage: make run ARGS="--serverAddr localhost:3001 --clientId test01")
run:
	CGO_ENABLED=0 go run . $(ARGS)

# Run with default test settings
run-local:
	CGO_ENABLED=0 go run . --serverAddr localhost:3001 --clientId virtual --httpPort 8080

# Run against the dev OCPP server (set DEV_SERVER_ADDR in .env)
run-dev:
	@test -n "$(DEV_SERVER_ADDR)" || (echo "DEV_SERVER_ADDR is not set, see .env.example" && exit 1)
	CGO_ENABLED=0 go run . --serverAddr $(DEV_SERVER_ADDR) --clientId virtual --httpPort 8080

# Clean build artifacts
clean:
	rm -f simulator
