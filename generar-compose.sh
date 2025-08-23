#!/bin/bash
OUTFILE="$1"
NUM_CLIENTS="$2"
python3 generate_clients.py "$NUM_CLIENTS" "$OUTFILE"