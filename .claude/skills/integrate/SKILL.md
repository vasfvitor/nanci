---
name: integrate
description: Use when bringing a finished lane (subagent worktree commits) into nanci's main or an integration branch. Cherry-pick or fast-forward, resolve doc conflicts, run the full verification, check LF and signatures, clean the worktree, and hand the push to the owner.
---

# Integrate: land a lane

## Steps

1. **Read the report, then the diff.** `git show --stat <sha>` for each commit; open anything that touches schema, migrations, bindings, `go.mod` or security code. Reject a commit that mixes an unrelated fix with its subject.
2. **Land it.** On the integration branch (clean tree): `git merge --ff-only <worktree-branch>` when the lane was based on the tip, else `git cherry-pick <shas>`. Cherry-pick lanes that touch `client.ts`/`types/desktop.ts`/`wailsjs` one at a time, never in parallel.
3. **Conflicts.** Expect them only in docs edited by two lanes (`docs/NFSE_ADN.md` happened). Resolve by keeping both edits; never drop a lane's sentence silently. Write the resolved file with LF.
4. **Verify the integrated tree**, not just the lane: Go root tests + lint, desktop build + tests, frontend `lint:check`, `test:unit`, `build`, `govulncheck` when deps moved. The lane's own run proves the lane; this run proves the combination.
5. **Line endings.** `file <changed files>` must say no CRLF. Python edits on Windows must open with `newline='\n'`; prefer `sed -i` or the Edit tool.
6. **Signatures.** `git log --format=%G? origin/main..HEAD | sort | uniq -c` must be only `G`. Any history rewrite (`filter-branch`, `rebase`, `commit --amend` of someone else's commit) drops signatures: re-sign with `git rebase --exec 'git commit --amend --no-edit -S' <base>` and confirm the tree is unchanged (`git diff <old-tip> HEAD` empty).
7. **Messages.** No attribution trailers. If one slipped in: `git filter-branch -f --msg-filter 'sed "/^Claude-Session:/d"' <base>..HEAD`, then step 6 again.
8. **Clean up.** `git worktree remove --force <path>` (if "Filename too long", `rm -rf` the directory then `git worktree prune`), delete the `worktree-agent-*` branch.
9. **Roadmap.** Tick the item in `docs/ROADMAP.md` (one line with the date, do not rewrite the section); remove resolved items from `docs/PENDENCIAS.md`. Commit as `docs: …`.
10. **Hand off.** Maintenance batches go to `main` as individual commits (`git merge --ff-only`), no PR, no squash. `git push` is blocked for Claude: give the owner the exact command and the count of commits, then watch `gh run list` for the CI result after they push.

## What to report

Commits landed (subject per line), verification results, bugs found but not fixed, and what is left for the owner (push, issue comments, decisions).
