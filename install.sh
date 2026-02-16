#!/bin/bash
#
# Myrai Installer
# https://github.com/gmsas95/goclawde-cli
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/gmsas95/goclawde-cli/main/install.sh | bash
#   or
#   curl -fsSL https://myrai.ai/install.sh | bash
#

set -e

BINARY_NAME="myrai"
REPO="gmsas95/goclawde-cli"
INSTALL_DIR="/usr/local/bin"
VERSION="${VERSION:-latest}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_logo() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                                                                  ║"
    echo "║                    🤖 Myrai Installer                            ║"
    echo "║                                                                  ║"
    echo "║              Your Personal AI Assistant                          ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

get_os() {
    case "$(uname -s)" in
        Darwin*)    echo "darwin" ;;
        Linux*)     echo "linux" ;;
        CYGWIN*|MINGW*|MSYS*)    echo "windows" ;;
        *)          echo "linux" ;;
    esac
}

get_arch() {
    case "$(uname -m)" in
        x86_64|amd64)    echo "amd64" ;;
        arm64|aarch64)   echo "arm64" ;;
        arm*)            echo "arm" ;;
        *)               echo "amd64" ;;
    esac
}

get_latest_version() {
    if command -v curl &> /dev/null; then
        curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget &> /dev/null; then
        wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        echo ""
    fi
}

download_file() {
    local url="$1"
    local output="$2"

    if command -v curl &> /dev/null; then
        curl -fsSL "$url" -o "$output"
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O "$output"
    else
        echo -e "${RED}Error: Neither curl nor wget is available${NC}"
        exit 1
    fi
}

install_binary() {
    local os=$(get_os)
    local arch=$(get_arch)
    local ext=""
    
    if [ "$os" = "windows" ]; then
        ext=".exe"
    fi

    if [ "$VERSION" = "latest" ]; then
        VERSION=$(get_latest_version)
        if [ -z "$VERSION" ]; then
            echo -e "${YELLOW}Could not determine latest version, using 'dev'${NC}"
            VERSION="dev"
        fi
    fi

    # Remove 'v' prefix if present
    VERSION="${VERSION#v}"

    local filename="${BINARY_NAME}-${os}-${arch}${ext}"
    local download_url="https://github.com/${REPO}/releases/download/v${VERSION}/${filename}"

    echo -e "${BLUE}Downloading Myrai ${VERSION} for ${os}/${arch}...${NC}"
    
    local tmp_dir=$(mktemp -d)
    local tmp_file="${tmp_dir}/${BINARY_NAME}${ext}"

    if ! download_file "$download_url" "$tmp_file"; then
        echo -e "${RED}Failed to download from: ${download_url}${NC}"
        echo -e "${YELLOW}Trying alternative URL...${NC}"
        
        # Try without version prefix
        download_url="https://github.com/${REPO}/releases/download/${VERSION}/${filename}"
        if ! download_file "$download_url" "$tmp_file"; then
            echo -e "${RED}Failed to download Myrai${NC}"
            echo -e "${YELLOW}You can build from source:${NC}"
            echo "  git clone https://github.com/${REPO}.git"
            echo "  cd goclawde-cli"
            echo "  make build"
            exit 1
        fi
    fi

    chmod +x "$tmp_file"

    # Determine install directory
    if [ -w "/usr/local/bin" ]; then
        INSTALL_DIR="/usr/local/bin"
    elif [ -w "$HOME/.local/bin" ]; then
        INSTALL_DIR="$HOME/.local/bin"
    else
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
    fi

    local target="${INSTALL_DIR}/${BINARY_NAME}${ext}"

    echo -e "${BLUE}Installing to ${target}...${NC}"
    
    if [ -w "$(dirname "$target")" ]; then
        mv "$tmp_file" "$target"
    else
        echo -e "${YELLOW}Requires sudo to install to ${INSTALL_DIR}${NC}"
        sudo mv "$tmp_file" "$target"
    fi

    rm -rf "$tmp_dir"

    echo -e "${GREEN}✓ Installed Myrai to ${target}${NC}"
}

check_path() {
    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        echo ""
        echo -e "${YELLOW}⚠️  ${INSTALL_DIR} is not in your PATH${NC}"
        echo ""
        echo "Add it to your shell config:"
        echo ""
        echo "  For bash (~/.bashrc):"
        echo "    export PATH=\"\$PATH:${INSTALL_DIR}\""
        echo ""
        echo "  For zsh (~/.zshrc):"
        echo "    export PATH=\"\$PATH:${INSTALL_DIR}\""
        echo ""
        echo "Then run: source ~/.bashrc  (or source ~/.zshrc)"
    fi
}

verify_installation() {
    echo ""
    echo -e "${BLUE}Verifying installation...${NC}"
    
    if command -v myrai &> /dev/null; then
        local version=$(myrai version 2>/dev/null || echo "installed")
        echo -e "${GREEN}✓ Myrai ${version} is ready!${NC}"
    else
        echo -e "${YELLOW}Run 'hash -r' or start a new terminal to use myrai${NC}"
    fi
}

print_next_steps() {
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                    Installation Complete!                        ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BLUE}Next Steps:${NC}"
    echo ""
    echo "  1. Run the setup wizard:"
    echo "     ${GREEN}myrai onboard${NC}"
    echo ""
    echo "  2. Start using Myrai:"
    echo "     ${GREEN}myrai --help${NC}"
    echo "     ${GREEN}myrai --cli${NC}"
    echo "     ${GREEN}myrai server${NC}"
    echo ""
    echo "  3. Documentation:"
    echo "     https://github.com/${REPO}#readme"
    echo ""
}

main() {
    print_logo

    # Check for curl or wget
    if ! command -v curl &> /dev/null && ! command -v wget &> /dev/null; then
        echo -e "${RED}Error: Either curl or wget is required${NC}"
        exit 1
    fi

    # Parse arguments
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --version|-v)
                VERSION="$2"
                shift 2
                ;;
            --dir|-d)
                INSTALL_DIR="$2"
                shift 2
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --version, -v VERSION    Install specific version (default: latest)"
                echo "  --dir, -d DIRECTORY      Install directory (default: /usr/local/bin)"
                echo "  --help, -h               Show this help"
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    install_binary
    check_path
    verify_installation
    print_next_steps
}

main "$@"
