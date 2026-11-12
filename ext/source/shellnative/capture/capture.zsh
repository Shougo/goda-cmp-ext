#!/usr/bin/env zsh

zmodload zsh/zpty || { echo 'error: missing module zsh/zpty' >&2; exit 1 }
zmodload zsh/datetime zsh/zselect || { echo 'error: missing module zsh/datetime or zsh/zselect' >&2; exit 1 }

# The output of the pty is read a line at a time, at most MAX_OUTPUT bytes
# for READ_TIMEOUT seconds: a pty that never gives what is waited for (or
# writes without end) must not hang nor take all the memory. (zpty -r -m
# reads until the pattern matches, without a limit.)
MAX_OUTPUT=$(( 8 * 1024 * 1024 ))
READ_TIMEOUT=${GODA_CMP_SHELLNATIVE_READ_TIMEOUT:-10}

# read_until_nuls: the output into $line up to its second null byte.
read_until_nuls() {
    local chunk
    local -F deadline=$(( EPOCHREALTIME + READ_TIMEOUT ))
    integer nuls=0
    line=
    while (( nuls < 2 )); do
        (( EPOCHREALTIME < deadline )) || { echo "error: pty read timeout" >&2; return 1 }
        if zpty -r -t z chunk; then
            line+=$chunk
            [[ $chunk == *$'\0'* ]] && (( nuls += ${#${chunk//[^$'\0']/}} ))
            (( $#line <= MAX_OUTPUT )) || { echo "error: pty output over $MAX_OUTPUT bytes" >&2; return 1 }
        elif ! zpty -t z; then
            echo "error: pty exited" >&2
            return 1
        else
            # Until it has more (or 0.1 seconds, for the deadline).
            zselect -t 10 -r $zpty_fd
        fi
    done
    return 0
}

# read_until_ok: the output up to "ok" at the end of a line (the rcfile is
# sourced), within 30 seconds (compinit).
read_until_ok() {
    local chunk
    local -F deadline=$(( EPOCHREALTIME + 30 ))
    line=
    while [[ $line != *ok$'\r'(|$'\n') ]]; do
        (( EPOCHREALTIME < deadline )) || return 1
        if zpty -r -t z chunk; then
            line+=$chunk
            (( $#line <= MAX_OUTPUT )) || return 1
            # Only the last line matters.
            [[ $chunk == *$'\n' ]] && [[ $line != *ok$'\r'$'\n' ]] && line=
        elif ! zpty -t z; then
            return 1
        else
            # Until it has more (or 0.1 seconds, for the deadline).
            zselect -t 10 -r $zpty_fd
        fi
    done
    return 0
}

local zpty_rcfile=${0:h}/zptyrc.zsh
[[ -r $zpty_rcfile ]] || { echo "error: rcfile not found: $zpty_rcfile" >&2; exit 1; }

# Spawn shell with non-blocking (-b) output
zpty -b z zsh --no-rcs --interactive
# (The file descriptor of the pty, waited on.)
zpty_fd=$REPLY

# Line buffer for pty output
local line=

# Initialize shell settings before processing
zpty -w z "source ${(qq)zpty_rcfile} && echo ok || exit 2"
read_until_ok || { echo "error: pty initialization failure" >&2; exit 2 }

# Constants
MSG_CHDIR="chdir:"
MSG_INPUT="input:"

END_OF_COMPLETION=$'\n\x01EOC\x01\n'

ACCEPT_LINE=$'\C-J'
COMPLETE_WORD=$'\C-I'
KILL_BUFFER=$'\C-U'
NULL_BYTE=$'\0'

# Main loop to read from stdin and process completion
local message=
local user_input=
while true; do
    IFS= read -r message || break

    case $message in
        $MSG_CHDIR*)
            # Change the current working directory in the pty
            local new_cwd=${message#$MSG_CHDIR}
            zpty -w -n z "${KILL_BUFFER}cd ${(qq)new_cwd}${ACCEPT_LINE}"
            continue
            ;;
        $MSG_INPUT*)
            # Do completion
            user_input=${message#$MSG_INPUT}
            ;;
        *)
            echo "error: invalid message: $message" >&2
            exit 1
            ;;
    esac

    # Trigger completion and send it to the pty
    zpty -w -n z "${KILL_BUFFER}${user_input}${COMPLETE_WORD}"

    # Completion results are output between null bytes
    # (The pty is in an unknown state after a failure: the source starts the
    # shell again.)
    read_until_nuls || { echo "error: pty read failure" >&2; exit 1 }
    echo -nE - "${${(@0)line}[2]}${END_OF_COMPLETION}"

    user_input=
done
