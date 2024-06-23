#!/bin/sh

if [ -f "${CUSTOM_WORKSPACE}/id_rsa" ]; then
  mkdir -p ~/.ssh
  cp "${CUSTOM_WORKSPACE}/id_rsa" ~/.ssh/id_rsa
  chmod 600 ~/.ssh/id_rsa
fi

# Execute the passed command
exec "$@"
