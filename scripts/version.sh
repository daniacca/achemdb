#!/bin/bash
# Version management script for AChemDB
# This script helps manage version tags for releases

set -e

VERSION_FILE="VERSION"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Get current version from VERSION file or git tag
get_version() {
    if [ -f "$REPO_ROOT/$VERSION_FILE" ]; then
        cat "$REPO_ROOT/$VERSION_FILE"
    else
        # Try to get version from latest git tag
        git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0"
    fi
}

# Set version in VERSION file
set_version() {
    local version="$1"
    echo "$version" > "$REPO_ROOT/$VERSION_FILE"
    echo "Version set to: $version"
}

# Show current version
show_version() {
    local version=$(get_version)
    echo "Current version: $version"
}

# Create a git tag for the current version
create_tag() {
    local version=$(get_version)
    local tag="v$version"
    
    if git rev-parse "$tag" >/dev/null 2>&1; then
        echo "Tag $tag already exists"
        exit 1
    fi
    
    git tag -a "$tag" -m "Release $tag"
    echo "Created tag: $tag"
    echo "Push with: git push origin $tag"
}

# Bump version
bump_version() {
    local part="$1"  # major, minor, patch
    local current=$(get_version)
    local major minor patch
    
    IFS='.' read -r major minor patch <<< "$current"
    
    case "$part" in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
        *)
            echo "Usage: $0 bump [major|minor|patch]"
            exit 1
            ;;
    esac
    
    local new_version="$major.$minor.$patch"
    set_version "$new_version"
    echo "Version bumped to: $new_version"
}

# Main command handler
case "${1:-show}" in
    show)
        show_version
        ;;
    set)
        if [ -z "$2" ]; then
            echo "Usage: $0 set <version>"
            exit 1
        fi
        set_version "$2"
        ;;
    bump)
        if [ -z "$2" ]; then
            echo "Usage: $0 bump [major|minor|patch]"
            exit 1
        fi
        bump_version "$2"
        ;;
    tag)
        create_tag
        ;;
    *)
        echo "Usage: $0 [show|set|bump|tag]"
        echo ""
        echo "Commands:"
        echo "  show              Show current version"
        echo "  set <version>     Set version (e.g., 1.0.0)"
        echo "  bump [major|minor|patch]  Bump version"
        echo "  tag               Create git tag for current version"
        exit 1
        ;;
esac

