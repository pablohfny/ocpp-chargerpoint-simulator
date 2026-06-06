.PHONY: build run clean

# Build the simulator
build:
	CGO_ENABLED=0 go build -o simulator .

# Run with arguments (usage: make run ARGS="--serverAddr localhost:3001 --clientId test01")
run:
	CGO_ENABLED=0 go run . $(ARGS)

# Run with default test settings
run-local:
	# CGO_ENABLED=0 go run . --serverAddr ws.star
	-ev.com --clientId virtual --httpPort 8080
	CGO_ENABLED=0 go run . --serverAddr localhost:3001 --clientId virtual --httpPort 8080

# Run with default test settings
run-dev:
	CGO_ENABLED=0 go run . --serverAddr nestjs-ocpp-server-development.up.railway.app --clientId virtual --httpPort 8080

# Clean build artifacts
clean:
	rm -f simulator
