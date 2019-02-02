# So we can see what commands get ran from the command line output.
SHELL = sh -xv

default: up

# Start up the chat app
.PHONY: up
up:
	docker-compose up -d

# Bring down the chat app
.PHONY: down
down:
	docker-compose down
