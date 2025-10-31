#!/bin/bash

VAULT_PATH=~/.local/share/vault/vault.db

# Function to show usage
show_usage() {
    echo "Usage: $0 [command] [args...]"
    echo "Available commands:"
    echo "  unlock - Unlock the vault"
    echo "  add [name] [username] - Add a new entry"
    echo "  list - List all entries"
    echo "  get [name] - Get an entry"
    exit 1
}

# Check if vault exists
if [ ! -f "$VAULT_PATH" ]; then
    echo "Vault not found at $VAULT_PATH"
    echo "Run './vault init' first to create a new vault"
    exit 1
fi

# Handle commands
case "$1" in
    unlock)
        VAULT_LOCK_TIMEOUT=5m ./vault --vault "$VAULT_PATH" unlock
        ;;
    add)
        if [ -z "$2" ] || [ -z "$3" ]; then
            echo "Usage: $0 add [name] [username]"
            exit 1
        fi
        VAULT_LOCK_TIMEOUT=5m ./vault --vault "$VAULT_PATH" add "$2" --username "$3" --secret-prompt
        ;;
    list)
        VAULT_LOCK_TIMEOUT=5m ./vault --vault "$VAULT_PATH" list
        ;;
    get)
        if [ -z "$2" ]; then
            echo "Usage: $0 get [name]"
            exit 1
        fi
        VAULT_LOCK_TIMEOUT=5m ./vault --vault "$VAULT_PATH" get "$2"
        ;;
    *)
        show_usage
        ;;
esac
