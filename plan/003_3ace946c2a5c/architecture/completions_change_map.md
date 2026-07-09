# Completions Change Map — Delta 003

Per-file exact edits for adding `completion` as a completable first-arg subcommand to all three shell completion files. Source: scout research of the live files.

## File 1: `completions/skilldozer.bash` (69 lines)

### Edit 1: Suppression walk (~line 52)
```bash
# CURRENT:
    [[ "${words[i]}" == "check" || "${words[i]}" == "init" ]] && return 0
# TARGET:
    [[ "${words[i]}" == "check" || "${words[i]}" == "init" || "${words[i]}" == "completion" ]] && return 0
```

### Edit 2: First-positional offer (~line 63)
```bash
# CURRENT:
    (( have_pos == 0 )) && cands="$cands check init"
# TARGET:
    (( have_pos == 0 )) && cands="$cands check init completion"
```

---

## File 2: `completions/_skilldozer` (zsh, 59 lines)

### Edit 1: First-positional compadd (~line 46)
```zsh
# CURRENT:
            compadd -- "$tags[@]" check init
# TARGET:
            compadd -- "$tags[@]" check init completion
```

### Edit 2: Suppression condition (~line 52)
```zsh
# CURRENT:
            if (( ${words[(I)check]} || ${words[(I)init]} )); then
# TARGET:
            if (( ${words[(I)check]} || ${words[(I)init]} || ${words[(I)completion]} )); then
```

---

## File 3: `completions/skilldozer.fish` (51 lines)

### Edit 1: New first-arg directive (after ~line 44)
```fish
# ADD after the `init` directive:
complete -c skilldozer -n '__fish_is_first_arg' -a 'completion' -d 'Emit the shell completion script for eval'
```

### Edit 2: Suppression predicate (~line 50)
```fish
# CURRENT:
complete -c skilldozer -n 'not __fish_seen_subcommand_from check init; and not __fish_prev_arg_in --search -s' \
# TARGET:
complete -c skilldozer -n 'not __fish_seen_subcommand_from check init completion; and not __fish_prev_arg_in --search -s' \
```

---

## Cross-file constraints
- Do NOT add `--shell` to any flag matrix (§14.1/§14.2: flag matrix is §6.1/§6.2 only)
- Do NOT change tag completion (`skilldozer --relative --all 2>/dev/null`)
- `completion` must be gated identically to `check`/`init` in all three files
- Validation: `grep -q 'completion' completions/skilldozer.bash && grep -q 'completion' completions/_skilldozer && grep -q 'completion' completions/skilldozer.fish`
