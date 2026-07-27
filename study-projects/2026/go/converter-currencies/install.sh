#!/usr/bin/env bash

set -e

REPO="https://github.com/goldchell-bra7/goldchell-bra7.git"
PROJECT="study-projects/2026/go/converter-currencies"
APP_NAME="converter-currencies"

GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
RESET='\033[0m'

echo -e "${GREEN}== ${APP_NAME} Installer ==${RESET}"
echo

echo -e "${BLUE}[1/6] Checking Git...${RESET}"
command -v git >/dev/null 2>&1 || {
    echo -e "${RED}Git is not installed.${RESET}"
    exit 1
}

echo -e "${BLUE}[2/6] Checking Go...${RESET}"
command -v go >/dev/null 2>&1 || {
    echo -e "${RED}Go is not installed.${RESET}"
    exit 1
}

TMP_DIR=$(mktemp -d)

echo -e "${BLUE}[3/6] Downloading project...${RESET}"
git clone --depth 1 "$REPO" "$TMP_DIR" >/dev/null

cd "$TMP_DIR/$PROJECT"

echo -e "${BLUE}[4/6] Downloading dependencies...${RESET}"
go mod tidy

echo -e "${BLUE}[5/6] Building...${RESET}"
go build -o "$APP_NAME"

echo -e "${BLUE}[6/6] Installing...${RESET}"
mkdir -p "$HOME/.local/bin"
mv "$APP_NAME" "$HOME/.local/bin/"

cd
rm -rf "$TMP_DIR"

echo
echo -e "${GREEN}Installation completed successfully!${RESET}"
echo
echo "Executable:"
echo "  $HOME/.local/bin/$APP_NAME"
echo
echo "If the command is not found, add this directory to your PATH:"
echo 'export PATH="$HOME/.local/bin:$PATH"'
