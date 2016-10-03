#!/bin/bash

echo "DEV_MODE $DEV_MODE" 1>&2

cat <<'EOF'
SECRET=$(porter_get_secrets)
[[ $SECRET = 'hi' ]] || exit 1
EOF
