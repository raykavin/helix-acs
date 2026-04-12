#!/bin/sh
set -e

# HELPERS
log() { echo "> $*"; }
info() { echo "  $*"; }

# GIT
log "Configuring Git"
if [ ! -d ".git" ]; then
  info "Initializing repository (branch: $GIT_INIT_DEFAULT_BRANCH)"
  git init --initial-branch="$GIT_INIT_DEFAULT_BRANCH"
else
  info "Repository already exists, skipping init"
fi
git config --global --add safe.directory "/workspaces/app"
git config --global user.name  "$GIT_CONFIG_DEV_USERNAME"
git config --global user.email "$GIT_CONFIG_DEV_EMAIL"
if git remote | grep -q "^origin$"; then
  CURRENT_URL="$(git remote get-url origin)"
  if [ "$CURRENT_URL" != "$GIT_REPO_ADDRESS" ]; then
    info "Updating origin URL"
    git remote set-url origin "$GIT_REPO_ADDRESS"
  else
    info "Origin already configured correctly"
  fi
else
  info "Adding origin remote"
  git remote add origin "$GIT_REPO_ADDRESS"
fi
if git show-ref --verify --quiet "refs/heads/$GIT_INIT_DEFAULT_BRANCH"; then
  git branch --set-upstream-to="origin/$GIT_INIT_DEFAULT_BRANCH" "$GIT_INIT_DEFAULT_BRANCH" 2>/dev/null || true
fi

# GO PRIVATE PROXY
if [ -n "$GO_PROXY_HOST" ] && [ -n "$GO_PROXY_PORT" ] && [ -n "$GIT_CONFIG_DEV_USERNAME" ] && [ -n "$GIT_CONFIG_TOKEN" ]; then
  log "Go private proxy env vars detected, running proxy-setup.sh"
  sh .devcontainer/proxy-setup.sh
else
  info "Go private proxy env vars not set, skipping proxy-setup.sh"
fi

# GO ENVIRONMENT
log "Adjusting Go directory permissions"
sudo chmod -R 777 /go/bin /go/pkg
log "Ensuring GOPATH is in PATH"
export PATH="$PATH:/go/bin"
log "Initializing Go modules"
if [ ! -f go.mod ]; then
  info "Creating module: $PROJECT_NAME"
  go mod init "$PROJECT_NAME"
else
  info "go.mod already exists, skipping"
fi
go mod tidy
if [ -n "$GOPRIVATE" ]; then
  log "Setting private Go repositories"
  go env -w GOPRIVATE="$GOPRIVATE"
fi

# GO TOOLS
log "Installing Go development tools"
go install golang.org/x/tools/gopls@latest
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
log "Verifying installed tools"
gopls version
dlv version
golangci-lint version
log "Running initial lint"
golangci-lint run || true

# BRANCH SETUP
log "Checking out develop branch"
if git show-ref --verify --quiet "refs/heads/develop"; then
  if git rev-parse --abbrev-ref HEAD | grep -q "^develop$"; then
    info "Already on develop branch"
  else
    info "Switching to develop branch"
    git checkout develop
  fi
fi

echo ""
log "Post-create completed successfully"
