# Issue tracker: GitHub

Issues and PRDs for this repository live in GitHub Issues at
`kriss-spy/bilidl`. Use the `gh` CLI for all operations and pass
`--repo kriss-spy/bilidl` explicitly.

## Conventions

- **Create an issue**: `gh issue create --repo kriss-spy/bilidl --title "..." --body "..."`. Use a heredoc for multi-line bodies.
- **Read an issue**: `gh issue view <number> --repo kriss-spy/bilidl --comments`, filtering comments with `jq` and fetching labels when needed.
- **List issues**: `gh issue list --repo kriss-spy/bilidl --state open --json number,title,body,labels,comments --jq '[.[] | {number, title, body, labels: [.labels[].name], comments: [.comments[].body]}]'`, with appropriate `--label` and `--state` filters.
- **Comment on an issue**: `gh issue comment <number> --repo kriss-spy/bilidl --body "..."`.
- **Apply or remove labels**: `gh issue edit <number> --repo kriss-spy/bilidl --add-label "..."` or `--remove-label "..."`.
- **Close an issue**: `gh issue close <number> --repo kriss-spy/bilidl --comment "..."`.

## When a skill says "publish to the issue tracker"

Create an issue in `kriss-spy/bilidl`.

## When a skill says "fetch the relevant ticket"

Run `gh issue view <number> --repo kriss-spy/bilidl --comments`.
