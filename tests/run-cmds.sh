#!/bin/bash
# OllamaTea Copyright (c) 2024 Neomantra Corp
# Runs simple examples of OllamaTea commands

set -e

echo -e "\033[31mHello\033[0m World" | ./bin/ot-ansi-to-png --out hello.png

./bin/ot-png-prompt --in ./tests/hello.png -m llava
