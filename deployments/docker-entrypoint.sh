#!/bin/sh
# docker-entrypoint.sh — Hera container startup.
#
# Runs hera init --ensure before starting hera-agent so that:
#   1. $HERA_HOME directory tree exists (first boot on a fresh volume).
#   2. Bundled skills are seeded into $HERA_HOME/skills/ (user-modified
#      skills are preserved across image upgrades).
#   3. Example config is present if the user has not created one yet.
#
# Uses exec so hera-agent replaces the shell and receives signals directly
# (Docker stop / Kubernetes SIGTERM propagates cleanly).
set -eu

# Seed skills and directory structure. Non-interactive; exits 0 on
# already-initialised, exits 1 only on real filesystem errors.
hera init --ensure

# Hand off to hera-agent (PID 1).
exec hera-agent "$@"
