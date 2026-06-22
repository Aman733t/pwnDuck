BINARY   = pwnduck
PI_HOST  = pi@$(PI)
DEPLOY   = /opt/pwnduck/pwnduck
UI_DIR   = ../pwnduck-ui/dist

.PHONY: build build-radxa build-local deploy deploy-full setup clean tidy

# Pi Zero W (ARMv6)
build:
	GOOS=linux GOARCH=arm GOARM=6 go build -ldflags="-s -w" -o $(BINARY) ./cmd/pwnduck
	@echo "Built for Pi Zero W (ARMv6)" && ls -lh $(BINARY)

# Radxa Zero (ARM64)
build-radxa:
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BINARY) ./cmd/pwnduck
	@echo "Built for Radxa Zero (ARM64)" && ls -lh $(BINARY)

# Mac (dev/testing)
build-local:
	go build -o $(BINARY)-mac ./cmd/pwnduck
	@echo "Built for local Mac" && ls -lh $(BINARY)-mac

# Deploy binary only + restart service
deploy: build
	@test -n "$(PI)" || (echo "Usage: make deploy PI=192.168.x.x" && exit 1)
	scp $(BINARY) $(PI_HOST):$(DEPLOY)
	ssh $(PI_HOST) "sudo chmod +x $(DEPLOY) && sudo systemctl restart pwnduck"
	@echo "Deployed to $(PI_HOST)"

# Full deploy — binary + library + UI
deploy-full: build
	@test -n "$(PI)" || (echo "Usage: make deploy-full PI=192.168.x.x" && exit 1)
	@echo "Deploying binary..."
	scp $(BINARY) $(PI_HOST):$(DEPLOY)
	@echo "Deploying library..."
	scp -r library/* $(PI_HOST):/opt/pwnduck/library/
	@echo "Deploying UI..."
	scp -r $(UI_DIR)/* $(PI_HOST):/opt/pwnduck/www/
	@echo "Restarting service..."
	ssh $(PI_HOST) "sudo chmod +x $(DEPLOY) && sudo systemctl restart pwnduck"
	@echo "Done! Open http://10.0.0.1:1337"

# Run setup script on Pi
setup:
	@test -n "$(PI)" || (echo "Usage: make setup PI=192.168.x.x" && exit 1)
	scp setup.sh $(PI_HOST):/home/pi/setup.sh
	ssh $(PI_HOST) "sudo bash /home/pi/setup.sh"

clean:
	rm -f $(BINARY) $(BINARY)-mac

tidy:
	go mod tidy