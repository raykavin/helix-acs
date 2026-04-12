# Enable auto-cd
set -g fish_auto_cd 1

# Golang path
set -x GOPATH /go
set -x GOROOT /usr/local/go
set -x PATH $PATH /go/bin

# Prompt
if status --is-interactive
    tide configure --auto
end

# Directory shortcuts
alias ..="cd .."
alias ...="cd ../.."
alias c="clear"

# Git
alias gst="git status"
alias gaa="git add ."
alias gc="git commit -m"
alias gp="git push"
alias gl="git log --oneline --graph --decorate"

# Go
alias gor="go run ."
alias got="go test ./..."
alias gob="go build"

# Z (for fast navigation)
z add (pwd)
