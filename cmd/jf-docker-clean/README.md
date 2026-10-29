# jf-docker-clean

Cleans up Docker tags in an Artifactory repository. It runs in dry-run mode by
default: it only prints what would be deleted. Deletion is enabled by `--delete`.

## Building

```sh
make build BINARY_NAME=jf-docker-clean
# or
go build -o bin/jf-docker-clean ./cmd/jf-docker-clean
```

## Configuration

Prefer environment variables for credentials (a token passed as a flag is
visible in `ps`):

```sh
export JFROG_URL=https://example.jfrog.io   # without /artifactory
export JFROG_ACCESS_TOKEN=...
export JFROG_USER=...   # optional
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--url` | `$JFROG_URL` | Artifactory base URL |
| `--token` | `$JFROG_ACCESS_TOKEN` | access token |
| `--user` | `$JFROG_USER` | username (optional) |
| `--repo` | `docker-local` | Docker repository |
| `--image` | – (required) | image name pattern (`*` and `?` wildcards, otherwise exact) |
| `--tag` | – (required) | tag pattern (`*` and `?` wildcards, otherwise exact) |
| `--protect` | `^v?\d+\.\d+\.\d+$` | regex of tags that are never deleted (`^$` disables protection) |
| `--keep-latest` | `5` | number of newest tags of each image to keep (`0` disables) |
| `--older-than-days` | `7` | delete only tags older than N days (`0` disables) |
| `--threads` | `2` | number of parallel deletions |
| `--delete` | off | actually delete (otherwise dry-run) |

`--image` and `--tag` are plain strings: without wildcards they must match
exactly, `*` matches any sequence of characters (including `/` in image names) and `?`
any single character. Both flags are required and must not be empty;
use `'*'` to match everything. `--protect` is the only option that takes a regex.

A tag is deleted only if it meets all conditions: it matches `--image` and
`--tag`, does not match `--protect`, is not among the `--keep-latest` newest
tags of its image, and is older than `--older-than-days`. Protected tags do not
count towards `--keep-latest`.

## Output

Every image affected by the rules is printed with its creation time and the
number of downloads (pulls) of the tag. `DELETE` lines are the ones that would be (or,
with `--delete`, are) removed, followed by `KEEP` lines: the newest images kept
only because of `--keep-latest`. Images protected by `--protect` or too new for
`--older-than-days` are not listed.

On a terminal `DELETE` is printed in red and `KEEP` in green. Colors are off when
the output is piped or redirected, or when `NO_COLOR` is set.

```
DELETE	docker-local/team/app:feature-x	2025-11-02T10:15:00Z	3 downloads
KEEP	docker-local/team/app:feature-y	2025-12-01T08:00:00Z	12 downloads
2 to delete, 1 kept by --keep-latest, 40 found
```

## Examples

Dry-run with defaults (keep the 5 newest, only tags older than 7 days):

```sh
jf-docker-clean --image '*' --tag '*'
```

A single image:

```sh
jf-docker-clean --image 'team/app' --tag '*'
```

Images with a prefix, only branch tags, keep the 10 newest:

```sh
jf-docker-clean --image 'team/*' --tag 'feature-*' --keep-latest 10
```

Images with a prefix, only `-dev.` pre-release tags (e.g. `1.2.3-dev.10`); plain
versions like `1.2.3` are protected by default and never touched:

```sh
jf-docker-clean --image 'team/app-*' --tag '*-dev.*'
```

Tags older than 30 days, without keeping the N newest:

```sh
jf-docker-clean --image '*' --tag '*' --keep-latest 0 --older-than-days 30
```

Additionally protect `latest` next to semver tags:

```sh
jf-docker-clean --image '*' --tag '*' \
  --protect '^(latest|v?\d+\.\d+\.\d+)$'
```

Delete for real after reviewing the dry-run output (8 parallel deletions):

```sh
jf-docker-clean --image 'team/app' --tag '*' --threads 8 --delete
```

## Notes

- Always start with a dry-run and review the list before using `--delete`.
- Tags are discovered through `manifest.json` in the layout
  `<image>/<tag>/manifest.json`. A different repository layout (e.g. with
  `list.manifest.json`) may require changing the AQL query in `artifactory.go`.
- `--image` and `--tag` are put into the AQL query as one `path $match "<image>/<tag>"`.
  Because `*` also matches `/`, the result is filtered again locally for exact matching.
- The tool has not been tested against a real Artifactory instance.
