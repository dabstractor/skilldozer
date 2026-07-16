# Bash completion for skilldozer.
#
# Install (one of):
#   source /path/to/skilldozer/completions/skilldozer.bash
#   cp completions/skilldozer.bash ~/.local/share/bash-completion/completions/skilldozer
#   cp completions/skilldozer.bash /etc/bash_completion.d/skilldozer
#
# Tags are derived DYNAMICALLY from disk by calling `skilldozer --relative --all`
# (skilldozer is manifest-free, PRD §2.1: there is no sidecar catalog to read).
#
# LOCKSTEP: the flag set below is frozen to `main.go parseArgs()`. If a future
# task adds/renames a flag there, update this list — and the zsh/fish files —
# identically. There is no shared source of truth the shells can import.
# Flags are long-form-only (decision 20): short aliases stay valid at runtime
# but are not advertised. Updated for --check/--init/--completions (decision 19):
# these were promoted from bare subcommands so the bare positional namespace
# belongs to skill tags — a bare <tab> shows skills, never commands.
# --shell's value completes to the bash/zsh/fish enum (§14.2); --shell is
# advertised (D7) since it is a real, documented flag in usageText OPTIONS.
_skilldozer_completion() {
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

    # Value-taking flags: route the value slot away from tag completion.
    #   --search        -> free-text query  -> offer NOTHING (return 0 with empty COMPREPLY).
    #   --store/--init  -> directory value  -> complete DIRECTORIES via compgen -d.
    #   --link          -> directory value  -> complete DIRECTORIES via compgen -d (§8.4).
    #   --shell         -> fixed enum       -> offer "bash zsh fish" via compgen -W.
    # (--store/--init/--link WANT path completion, unlike --search's free-text -> nothing.)
    case "$prev" in
        --search) return 0 ;;
        --store|--init|--link) COMPREPLY=($(compgen -d -- "$cur")); return 0 ;;
        --shell) COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur")); return 0 ;;
    esac

    # Flag completion when the current token starts with '-' (long-form only — decision 20).
    if [[ "$cur" == -* ]]; then
        COMPREPLY=($(compgen -W \
            "--version --help --path --list --all --file --relative --no-color --search --store --shell --check --init --link --completions" \
            -- "$cur"))
        return 0
    fi

    # Tags straight from the binary (canonical relTags, one per line). Errors
    # swallowed: a missing/broken skilldozer degrades to "no tags" instead of spewing
    # into the completion menu. Positionals are ALWAYS skills (decision 19: no bare
    # subcommands), and skills are never mutually exclusive — offer them on every
    # positional <tab>, first or later.
    local tags
    tags=$(skilldozer --relative --all 2>/dev/null)
    # SC2207 (mapfile preferred) is acceptable here: tags and flags never
    # contain spaces, so word-splitting is safe.
    COMPREPLY=($(compgen -W "$tags" -- "$cur"))
    return 0
}
complete -F _skilldozer_completion skilldozer

# --- §14.7 listing behavior (decision 22) -------------------------------------
# skilldozer wants every ambiguous match listed on the FIRST Tab — a manifest-free
# store (PRD §2) makes completion the primary discovery path, so candidates hidden
# behind a silent common-prefix halt are a UX defect. bash defaults to
# show-all-if-ambiguous OFF: the first Tab completes the common prefix and beeps,
# and the full list appears only on the second Tab.
#
# The line below sets show-all-if-ambiguous ON so all prefix matches list on the
# first Tab. This is a READLINE SESSION-GLOBAL option: it changes listing for EVERY
# command in this shell, not just skilldozer (there is no per-command scope). The
# `[[ $- == *i* ]] &&` guard keeps this quiet when the file is sourced
# non-interactively (e.g. an eval test harness): `bind` in a non-interactive shell
# prints a warning, which the guard silences. Completions only matter interactively,
# so the option still applies where it counts. The trailing `|| true` ensures
# `source`/`eval` of this file returns exit 0 even in a non-interactive shell, where
# the `[[ ... ]]` test is false and the `&&` would otherwise short-circuit the whole
# statement (and thus the source/eval) to exit 1 — which would abort a `.bashrc`
# running under `set -e` or break `source ... && cmd` one-liners.
{ [[ $- == *i* ]] && bind 'set show-all-if-ambiguous on'; } || true
# Opt-out — restore bash's stock (second-Tab) listing:
#   bind 'set show-all-if-ambiguous off'
