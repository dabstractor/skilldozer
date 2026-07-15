# Fish completion for skilldozer.
#
# Install:
#   cp completions/skilldozer.fish ~/.config/fish/completions/skilldozer.fish
#
# Tags are derived DYNAMICALLY from disk by calling `skilldozer --relative --all`
# (skilldozer is manifest-free, PRD §2.1: there is no sidecar catalog to read).
#
# LOCKSTEP: the flag list below is frozen to `main.go parseArgs()`. If a future
# task adds/renames a flag there, update this file — and the bash/zsh files —
# identically. There is no shared source of truth the shells can import.
# Flags are long-form-only (decision 20): short aliases stay valid at runtime
# but are not advertised. Updated for --check/--init/--completions (decision 19):
# these were promoted from bare subcommands so the bare positional namespace
# belongs to skill tags — a bare <tab> shows skills, never commands.
# --shell's value completes to the bash/zsh/fish enum (§14.2); --shell is
# advertised (D7) since it is a real, documented flag in usageText OPTIONS.

# No file completion: skilldozer takes tags/flags, not paths.
complete -c skilldozer -f

# Flag matrix (§6.1/§6.2). --relative and --no-color have NO short forms.
complete -c skilldozer -l version  -d 'Print the skilldozer version'
complete -c skilldozer -l help     -d 'Show this help message'
complete -c skilldozer -l path     -d 'Print the resolved skills directory'
complete -c skilldozer -l list     -d 'Human-readable catalog (TAG, NAME, DESCRIPTION)'
complete -c skilldozer -l all      -d 'Print every skill directory path, sorted by tag'
complete -c skilldozer -l file     -d 'Print the SKILL.md path instead of the directory'
complete -c skilldozer -l relative -d 'Print paths relative to the skills directory'
complete -c skilldozer -l no-color -d 'Disable ANSI color'
# Decision 19: check/init/completions promoted from bare subcommands to flags.
# Decision 20: long-form-only — no short aliases are advertised.
complete -c skilldozer -l check       -d 'Validate every skill on disk'
# --init <dir> (§8.2): like --store, its value is a directory; `-r` routes
# the value slot to file/dir completion (the inverse of --search's nothing).
complete -c skilldozer -l init        -d 'First-run setup: pick/create the skills store' -r
complete -c skilldozer -l completions -d 'Emit the shell completion script for eval'
# --search takes a free-text query, so NO completion is offered after it.
# We deliberately do NOT pass -r here: in fish 4.x `-r` switches into
# "complete the option's value" mode, which BYPASSES the global `-f` above and
# offers file names for the query. Without -r, --search is treated as a plain
# flag, so after `--search ` the global `-f` (no-files) applies and nothing is
# offered -- exactly the PRD §6.1 free-text-query behavior. (fish's -r is only a
# completion hint; skilldozer itself enforces that --search needs a value, exit 1.)
complete -c skilldozer -l search -d 'Substring search over tag/name/description/keywords'

# --store <dir> (PRD §8.2): Non-interactive store path for init. Unlike --search,
# --store's value is a DIRECTORY, so here we DO pass `-r`: in fish 4.x `-r`
# switches into "complete the option's value" mode, which BYPASSES the global
# `-f` above and offers file/dir paths for the value. This is the intentional
# INVERSE of --search's no-`-r` (free-text -> offer nothing). No short form.
complete -c skilldozer -l store -d 'Non-interactive store path for init' -r
# --shell <name> (PRD §14.2): Force a shell for completion. The value is a FIXED
# enum (bash/zsh/fish), so use `-x` (exclusive: require a value, NO file
# completion) + `-a "bash zsh fish"` (the three candidates). This is the THIRD
# value-routing pattern: --search = nothing (no flag), --store/--init = files
# (-r), --shell = closed enum (-x -a). --shell is advertised (decision D7).
complete -c skilldozer -l shell -d 'Force a shell for completion' -x -a "bash zsh fish"

# Dynamic tags: ONE directive with command substitution (NOT a hardcoded line per
# tag — the store is manifest-free and changes as skills are added). Suppressed
# only when the previous arg is --search (free-text query — no tag completion
# there). No subcommand guard: positionals are ALWAYS skills (decision 19).
complete -c skilldozer -n 'not __fish_prev_arg_in --search' \
    -a '(skilldozer --relative --all 2>/dev/null)' -d 'skill tag'
