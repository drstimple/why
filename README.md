# why v0.6.1 — just for fun

Experimental, evidence-first Unix diagnostics for macOS and Linux. Go 1.22+ and `ps` required. No third-party Go dependencies. Read-only diagnostics, except that `--not` initiates an outbound TCP connection to the requested target.

## Build, test, install

```sh
make test
make build
make install PREFIX="$HOME/.local"   # install without sudo
export PATH="$HOME/.local/bin:$PATH"
export MANPATH="$HOME/.local/share/man:${MANPATH:-}"
man why
```

Or use `sudo make install` to install to `/usr/local` (depending on your system's permissions). `man -l ./why.1` previews the manual on systems supporting `-l`.

## Examples

```sh
why --version              # print installed version
why 1
why --shell                 # inspect the process that invoked why
why --shell --because       # evidence ledger for the invoking process
why $$                      # inspect the current shell explicitly
why --because 1
why --sample 1s --json 1
why --not tcp:localhost:5432
why --not 'tcp:[::1]:5432' --timeout 500ms --json
```

`--not` exit codes: 0=unmet (connection refused), 1=met, 2=invalid input, 3=unknown. Process mode: 0=reported, 1=inspection error, 2=invalid input.

The command does **not** infer a root cause from a parent PID, a `ps` CPU percentage, or a TCP handshake. See `man why` for limitations.

## An entirely unnecessary but essential feature

```sh
why --why
why --why --json
why --why --why
why --why --why --json
```

No monetization. Just curiosity.

## Shell investigation

`why --shell` inspects the immediate parent PID of the `why` process. In an ordinary interactive shell, that is usually your shell. In a script, wrapper, or pipeline it may instead be another process. `why $$` is a shell-expanded alternative; `$$` has shell-specific behavior, particularly in subshells. Both approaches use best-effort `ps` snapshots, not audited process creation events.
