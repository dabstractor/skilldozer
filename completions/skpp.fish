# Fish completion for skpp.
#
# Install:
#   cp completions/skpp.fish ~/.config/fish/completions/skpp.fish
#
# Tags are derived DYNAMICALLY from disk by calling `skpp --relative --all`
# (skpp is manifest-free, PRD §2.1: there is no sidecar catalog to read).
#
# LOCKSTEP: the flag list below is frozen to `main.go parseArgs()`. If a future
# task adds/renames a flag there, update this file — and the bash/zsh files —
# identically.

# No file completion: skpp takes tags/flags, not paths.
complete -c skpp -f

# Flag matrix (§6.1/§6.2). --relative and --no-color have NO short forms.
complete -c skpp -s v -l version  -d 'Print the skpp version'
complete -c skpp -s h -l help     -d 'Show this help message'
complete -c skpp -s p -l path     -d 'Print the resolved skills directory'
complete -c skpp -s l -l list     -d 'Human-readable catalog (TAG, NAME, DESCRIPTION)'
complete -c skpp -s a -l all      -d 'Print every skill directory path, sorted by tag'
complete -c skpp -s f -l file     -d 'Print the SKILL.md path instead of the directory'
complete -c skpp       -l relative -d 'Print paths relative to the skills directory'
complete -c skpp       -l no-color -d 'Disable ANSI color'
# --search/-s take a free-text query, so NO completion is offered after them.
# We deliberately do NOT pass -r here: in fish 4.x `-r` switches into
# "complete the option's value" mode, which BYPASSES the global `-f` above and
# offers file names for the query. Without -r, --search/-s are treated as plain
# flags, so after `--search ` the global `-f` (no-files) applies and nothing is
# offered -- exactly the PRD §6.1 free-text-query behavior. (fish's -r is only a
# completion hint; skpp itself enforces that --search needs a value, exit 1.)
complete -c skpp -s s -l search -d 'Substring search over tag/name/description/keywords'

# `check` is an EXCLUSIVE subcommand (PRD §6.3). Offer it only as the first arg.
complete -c skpp -n '__fish_is_first_arg' -a 'check' -d 'Validate every skill on disk'

# Dynamic tags: ONE directive with command substitution (NOT a hardcoded line per
# tag — the store is manifest-free and changes as skills are added). Suppressed
# once `check` is seen (exclusive subcommand, PRD §6.3) AND when the previous
# arg is --search/-s (free-text query — no tag completion there either).
complete -c skpp -n 'not __fish_seen_subcommand_from check; and not __fish_prev_arg_in --search -s' \
    -a '(skpp --relative --all 2>/dev/null)' -d 'skill tag'
