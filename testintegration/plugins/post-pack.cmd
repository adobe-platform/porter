#!/bin/bash -ex
env

if [[ -z "$OLD_STYLE_HOOK_ENV" ]]; then
	echo "OLD_STYLE_HOOK_ENV is broken"
	exit 1
fi
