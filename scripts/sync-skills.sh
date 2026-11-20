#!/usr/bin/env bash
# sync-skills.sh rebuilds .claude/skills from .agents/skills. The skills live
# in .agents/skills, which Codex, Gemini CLI and Antigravity read; Claude Code
# reads only .claude/skills, and a symlink is checked out as a plain file on
# Windows, so Claude gets a generated copy. With --check, nothing is written
# and the exit code is 1 when the copy is out of date.
set -eu
cd "$(dirname "$0")/.."

src=.agents/skills
dst=.claude/skills

# build writes the copy of every skill in $src into the directory $1. The
# notice goes right after the front matter: Claude Code reads the front
# matter only when the opening --- is the first line.
build() {
	local out=$1 dir name
	mkdir -p "$out"
	for dir in "$src"/*/; do
		name=$(basename "$dir")
		cp -R "$dir" "$out/$name"
		awk -v name="$name" '
			{ print }
			/^---$/ && ++fences == 2 {
				print "<!-- Generated from .agents/skills/" name "/ by scripts/sync-skills.sh; edit there. -->"
			}
		' "$dir/SKILL.md" >"$out/$name/SKILL.md"
	done
}

if [ "${1:-}" = "--check" ]; then
	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT
	build "$tmp/skills"
	if ! diff -r "$tmp/skills" "$dst" >/dev/null 2>&1; then
		diff -r "$tmp/skills" "$dst" | head -20 || true
		echo "    .claude/skills não bate com .agents/skills: edite em .agents/skills e rode scripts/sync-skills.sh"
		exit 1
	fi
	exit 0
fi

rm -rf "$dst"
build "$dst"
