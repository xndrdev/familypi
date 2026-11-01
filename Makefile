# Baut Frontend + Backend und packt alles fuer den Raspberry Pi (64-bit Raspberry Pi OS) zusammen.
PI_HOST ?= pi@familypi.local
PI_DIR  ?= /opt/familydisplay
DIST    := dist

.PHONY: dev-backend dev-frontend test frontend backend-pi dist deploy clean

dev-backend:
	cd backend && FD_STATIC_DIR=../frontend/.output/public go run ./cmd/familydisplay

dev-frontend:
	cd frontend && pnpm dev

test:
	cd backend && go vet ./... && go test ./...
	cd frontend && pnpm exec nuxi typecheck

frontend:
	cd frontend && pnpm install --frozen-lockfile && pnpm generate

backend-pi:
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ../$(DIST)/familydisplay ./cmd/familydisplay

dist: frontend backend-pi
	rm -rf $(DIST)/public && cp -r frontend/.output/public $(DIST)/public
	cp deploy/familydisplay.service deploy/familydisplay-backup.service deploy/familydisplay-backup.timer deploy/kiosk.sh deploy/familydisplay.env.example $(DIST)/

deploy: dist
	rsync -av --delete --exclude data --exclude familydisplay.env $(DIST)/ $(PI_HOST):$(PI_DIR)/
	ssh $(PI_HOST) 'sudo systemctl restart familydisplay'

clean:
	rm -rf $(DIST) frontend/.output
