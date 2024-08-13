#!/bin/bash

# Define the plugin's Go project directory
# Assuming this script is run from the project directory
PROJECT_DIR="$(pwd)"

# Name of the plugin binary
BINARY_NAME="check-python-version"

# Define the path to the plugins directory
PLUGIN_DIR="$HOME/.foo/plugins"

# Ensure the Go environment is properly set
# Uncomment and set these if Go is not setup globally
# export GOROOT=/path/to/your/go/installation
# export GOPATH=/path/to/your/go/workspace
# export PATH=$PATH:$GOROOT/bin:$GOPATH/bin

echo "Building the plugin..."
# Navigate to the project directory and build the binary
cd "$PROJECT_DIR" || exit
go build -o "$BINARY_NAME"

# Check if the build was successful
if [ ! -f "$BINARY_NAME" ]; then
    echo "Build failed, binary not found."
    exit 1
fi

# Check if the plugins directory exists, create it if not
if [ ! -d "$PLUGIN_DIR" ]; then
    echo "Creating plugin directory at $PLUGIN_DIR"
    mkdir -p "$PLUGIN_DIR"
fi

# Deploy the binary to the plugins directory
echo "Deploying $BINARY_NAME to $PLUGIN_DIR"
cp "$BINARY_NAME" "$PLUGIN_DIR/"

# Verify and finish
if [ "$?" -eq 0 ]; then
    echo "Deployment successful."
else
    echo "Deployment failed."
    exit 1
fi
