# Bash completion for skpp.
#
# Install (one of):
#   source /path/to/skpp/completions/skpp.bash
#   cp completions/skpp.bash ~/.local/share/bash-completion/completions/skpp
#   cp completions/skpp.bash /etc/bash_completion.d/skpp
#
# Tags are derived DYNAMICALLY from disk by calling `skpp --relative --all`
# (skpp is manifest-free, PRD §2.1: there is no sidecar catalog to read).
#
# LOCKSTEP: the flag set below is frozen to `main.go parseArgs()`. If a future
# task adds/renames a flag there, update this list — and the zsh/fish files —
# identically. There is no shared source of truth the shells can import.
_skpp_completion() {
    local cur prev words cword
    # _init_completion (from the bash-completion package) sets cur/prev/words/cword.
    # Fall back to COMP_WORDS manually when the package is absent (minimal Linux,
    # macOS default bash) — otherwise `_init_completion || return` silently offers
    # NOTHING. SC2317 flags the fallback branch as "unreachable"; it is a false
    # positive (the branch runs whenever the helper is missing).
    _init_completion 2>/dev/null || {
        cur="${COMP_WORDS[COMP_CWORD]}"
        prev="${COMP_WORDS[COMP_CWORD-1]}"
        cword=$COMP_CWORD
        words=("${COMP_WORDS[@]}")
        COMPREPLY=()
    }

    # --search / -s take a free-text query: offer nothing (no tag completion).
    case "$prev" in
        --search|-s) return 0 ;;
    esac

    # Flag completion when the current token starts with '-'.
    if [[ "$cur" == -* ]]; then
        COMPREPLY=($(compgen -W \
            "--version -v --help -h --path -p --list -l --all -a --file -f --relative --no-color --search -s" \
            -- "$cur"))
        return 0
    fi

    # Walk earlier words: `check` is an EXCLUSIVE subcommand (PRD §6.3 —
    # check+tags → exit 2), so once it appears, offer nothing further. Track
    # whether any non-flag positional was seen so `check` is only ever offered
    # as the FIRST positional token.
    local i have_pos=0
    for ((i=1; i<cword; i++)); do
        [[ "${words[i]}" == "check" ]] && return 0
        [[ "${words[i]}" == -* ]] && continue
        have_pos=1
    done

    # Tags straight from the binary (canonical relTags, one per line). Errors
    # swallowed: a missing/broken skpp degrades to "no tags" instead of spewing
    # into the completion menu.
    local tags cands
    tags=$(skpp --relative --all 2>/dev/null)
    cands="$tags"
    (( have_pos == 0 )) && cands="$cands check"
    # SC2207 (mapfile preferred) is acceptable here: tags and flags never
    # contain spaces, so word-splitting is safe.
    COMPREPLY=($(compgen -W "$cands" -- "$cur"))
    return 0
}
complete -F _skpp_completion skpp
