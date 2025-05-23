#!/bin/bash

# Multi-Platform Build Script for IP Marketplace Backend
# Builds the application for multiple operating systems and architectures

set -e

# Configuration
APP_NAME="imi-backend"
MAIN_PACKAGE="./cmd/server"
BUILD_DIR="dist"
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION=$(go version | awk '{print $3}')

# Build flags
LDFLAGS="-w -s -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME} -X main.GoVersion=${GO_VERSION}"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Platform and architecture combinations
declare -a PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "linux/386"
    "windows/amd64"
    "windows/386"
    "darwin/amd64"
    "darwin/arm64"
    "freebsd/amd64"
    "freebsd/386"
)

# Docker platforms (subset for Docker multi-arch builds)
declare -a DOCKER_PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
)

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to get binary extension for platform
get_binary_extension() {
    local os=$1
    if [[ "$os" == "windows" ]]; then
        echo ".exe"
    else
        echo ""
    fi
}

# Function to build for a specific platform
build_platform() {
    local platform=$1
    local os=$(echo $platform | cut -d'/' -f1)
    local arch=$(echo $platform | cut -d'/' -f2)
    local binary_name="${APP_NAME}-${os}-${arch}$(get_binary_extension $os)"
    local output_path="${BUILD_DIR}/${binary_name}"
    
    print_status "Building for ${os}/${arch}..."
    
    # Set environment variables for cross-compilation
    export GOOS=$os
    export GOARCH=$arch
    export CGO_ENABLED=0
    
    # Build the binary
    if go build -ldflags "${LDFLAGS}" -o "${output_path}" "${MAIN_PACKAGE}"; then
        # Get file size
        local size=$(ls -lh "${output_path}" | awk '{print $5}')
        print_success "Built ${binary_name} (${size})"
        
        # Create checksums
        if command -v sha256sum >/dev/null 2>&1; then
            sha256sum "${output_path}" > "${output_path}.sha256"
        elif command -v shasum >/dev/null 2>&1; then
            shasum -a 256 "${output_path}" > "${output_path}.sha256"
        fi
        
        return 0
    else
        print_error "Failed to build for ${os}/${arch}"
        return 1
    fi
}

# Function to compress binaries
compress_binaries() {
    if [[ "$COMPRESS" != "true" ]]; then
        return 0
    fi
    
    print_status "Compressing binaries..."
    
    if command -v upx >/dev/null 2>&1; then
        for binary in ${BUILD_DIR}/${APP_NAME}-*; do
            if [[ -f "$binary" && ! "$binary" =~ \.(sha256|zip|tar\.gz)$ ]]; then
                print_status "Compressing $(basename $binary)..."
                upx --best --lzma "$binary" 2>/dev/null || print_warning "Failed to compress $(basename $binary)"
            fi
        done
    else
        print_warning "UPX not found. Skipping compression. Install UPX for smaller binaries."
    fi
}

# Function to create archives
create_archives() {
    if [[ "$ARCHIVE" != "true" ]]; then
        return 0
    fi
    
    print_status "Creating archives..."
    
    for binary in ${BUILD_DIR}/${APP_NAME}-*; do
        if [[ -f "$binary" && ! "$binary" =~ \.(sha256|zip|tar\.gz)$ ]]; then
            local basename=$(basename "$binary")
            local os_arch=$(echo "$basename" | sed "s/${APP_NAME}-//")
            
            # Create directory structure for archive
            local temp_dir="${BUILD_DIR}/temp/${APP_NAME}-${VERSION}-${os_arch}"
            mkdir -p "$temp_dir"
            
            # Copy files to temp directory
            cp "$binary" "$temp_dir/"
            cp "${binary}.sha256" "$temp_dir/" 2>/dev/null || true
            cp README.md "$temp_dir/" 2>/dev/null || true
            cp LICENSE "$temp_dir/" 2>/dev/null || true
            
            # Create archive based on platform
            if [[ "$os_arch" =~ windows ]]; then
                # Create ZIP for Windows
                (cd "${BUILD_DIR}/temp" && zip -r "../${APP_NAME}-${VERSION}-${os_arch}.zip" "${APP_NAME}-${VERSION}-${os_arch}")
            else
                # Create tar.gz for Unix-like systems
                (cd "${BUILD_DIR}/temp" && tar -czf "../${APP_NAME}-${VERSION}-${os_arch}.tar.gz" "${APP_NAME}-${VERSION}-${os_arch}")
            fi
            
            print_success "Created archive for ${os_arch}"
        fi
    done
    
    # Clean up temp directory
    rm -rf "${BUILD_DIR}/temp"
}

# Function to build Docker images for multiple architectures
build_docker_multiarch() {
    if [[ "$DOCKER" != "true" ]]; then
        return 0
    fi
    
    print_status "Building multi-architecture Docker images..."
    
    # Check if Docker buildx is available
    if ! docker buildx version >/dev/null 2>&1; then
        print_error "Docker buildx is required for multi-architecture builds"
        return 1
    fi
    
    # Create builder if it doesn't exist
    docker buildx create --name multiarch-builder --use 2>/dev/null || docker buildx use multiarch-builder
    
    # Build and push multi-arch image
    local docker_platforms=$(IFS=','; echo "${DOCKER_PLATFORMS[*]}")
    local image_tag="${DOCKER_REGISTRY:-localhost}/${APP_NAME}:${VERSION}"
    
    print_status "Building Docker image for platforms: ${docker_platforms}"
    
    docker buildx build \
        --platform "${docker_platforms}" \
        --build-arg VERSION="${VERSION}" \
        --build-arg COMMIT="${COMMIT}" \
        --build-arg BUILD_TIME="${BUILD_TIME}" \
        -t "${image_tag}" \
        -t "${DOCKER_REGISTRY:-localhost}/${APP_NAME}:latest" \
        ${DOCKER_PUSH:+--push} \
        .
    
    print_success "Docker multi-arch build completed"
}

# Function to generate build report
generate_report() {
    local report_file="${BUILD_DIR}/build-report.txt"
    
    print_status "Generating build report..."
    
    cat > "$report_file" << EOF
IP MARKETPLACE BACKEND - BUILD REPORT
=====================================

Build Information:
- Version: ${VERSION}
- Commit: ${COMMIT}
- Build Time: ${BUILD_TIME}
- Go Version: ${GO_VERSION}
- Builder: $(whoami)@$(hostname)

Built Binaries:
EOF
    
    for binary in ${BUILD_DIR}/${APP_NAME}-*; do
        if [[ -f "$binary" && ! "$binary" =~ \.(sha256|zip|tar\.gz|txt)$ ]]; then
            local size=$(ls -lh "$binary" | awk '{print $5}')
            local basename=$(basename "$binary")
            echo "- ${basename} (${size})" >> "$report_file"
        fi
    done
    
    if [[ "$ARCHIVE" == "true" ]]; then
        echo "" >> "$report_file"
        echo "Archives:" >> "$report_file"
        for archive in ${BUILD_DIR}/*.{zip,tar.gz}; do
            if [[ -f "$archive" ]]; then
                local size=$(ls -lh "$archive" | awk '{print $5}')
                local basename=$(basename "$archive")
                echo "- ${basename} (${size})" >> "$report_file"
            fi
        done
    fi
    
    echo "" >> "$report_file"
    echo "Total build time: ${build_duration}s" >> "$report_file"
    
    print_success "Build report saved to ${report_file}"
}

# Function to clean build directory
clean_build() {
    if [[ -d "$BUILD_DIR" ]]; then
        print_status "Cleaning previous build..."
        rm -rf "$BUILD_DIR"
    fi
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Build IP Marketplace Backend for multiple platforms

OPTIONS:
    -c, --clean         Clean build directory before building
    -z, --compress      Compress binaries with UPX
    -a, --archive       Create platform-specific archives
    -d, --docker        Build multi-architecture Docker images
    -p, --push          Push Docker images (requires --docker)
    -v, --version VER   Set version (default: git describe)
    -o, --output DIR    Set output directory (default: dist)
    -h, --help          Show this help message

ENVIRONMENT VARIABLES:
    VERSION             Build version
    COMMIT              Git commit hash
    DOCKER_REGISTRY     Docker registry for multi-arch builds
    DOCKER_PUSH         Set to 'true' to push Docker images

EXAMPLES:
    $0                          # Basic build for all platforms
    $0 -c -z -a                # Clean, build, compress, and archive
    $0 -d -p                   # Build and push Docker multi-arch images
    $0 -v v1.2.3 -o releases  # Build with custom version and output dir

PLATFORMS:
$(printf "    %s\n" "${PLATFORMS[@]}")

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--clean)
            CLEAN=true
            shift
            ;;
        -z|--compress)
            COMPRESS=true
            shift
            ;;
        -a|--archive)
            ARCHIVE=true
            shift
            ;;
        -d|--docker)
            DOCKER=true
            shift
            ;;
        -p|--push)
            DOCKER_PUSH=true
            shift
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -o|--output)
            BUILD_DIR="$2"
            shift 2
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Main execution
main() {
    local start_time=$(date +%s)
    
    print_status "Starting multi-platform build for ${APP_NAME}"
    print_status "Version: ${VERSION}"
    print_status "Commit: ${COMMIT}"
    print_status "Go Version: ${GO_VERSION}"
    echo
    
    # Clean if requested
    if [[ "$CLEAN" == "true" ]]; then
        clean_build
    fi
    
    # Create build directory
    mkdir -p "$BUILD_DIR"
    
    # Check if Go is installed
    if ! command -v go >/dev/null 2>&1; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check if main package exists
    if [[ ! -f "${MAIN_PACKAGE}/main.go" ]]; then
        print_error "Main package not found at ${MAIN_PACKAGE}"
        exit 1
    fi
    
    # Build for all platforms
    local success_count=0
    local total_count=${#PLATFORMS[@]}
    
    for platform in "${PLATFORMS[@]}"; do
        if build_platform "$platform"; then
            ((success_count++))
        fi
    done
    
    echo
    print_status "Build summary: ${success_count}/${total_count} platforms built successfully"
    
    if [[ $success_count -eq 0 ]]; then
        print_error "No binaries were built successfully"
        exit 1
    fi
    
    # Post-build operations
    compress_binaries
    create_archives
    build_docker_multiarch
    
    # Calculate build duration
    local end_time=$(date +%s)
    build_duration=$((end_time - start_time))
    
    # Generate build report
    generate_report
    
    echo
    print_success "Multi-platform build completed in ${build_duration}s"
    print_success "Output directory: ${BUILD_DIR}"
    
    # List built artifacts
    echo
    print_status "Built artifacts:"
    ls -lh "${BUILD_DIR}" | grep -v "^total" | awk '{print "  " $9 " (" $5 ")"}'
}

# Trap to clean up on exit
trap 'print_error "Build interrupted"; exit 1' INT TERM

# Run main function
main "$@"
