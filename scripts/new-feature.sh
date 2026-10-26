#!/usr/bin/env sh
# Scaffold a feature package from internal/feature/_template.
#   ./scripts/new-feature.sh invoice
set -eu

name="${1:?usage: new-feature.sh <name> (lowercase, singular, e.g. invoice)}"
case "$name" in
  *[!a-z0-9]* | [0-9]*) echo "name must be lowercase letters/digits, starting with a letter" >&2; exit 1 ;;
esac

src="internal/feature/_template"
dst="internal/feature/$name"
[ -e "$dst" ] && { echo "$dst already exists" >&2; exit 1; }

# Capitalise the first letter for type names: invoice -> Invoice.
Name="$(printf '%s' "$name" | cut -c1 | tr '[:lower:]' '[:upper:]')$(printf '%s' "$name" | cut -c2-)"

mkdir -p "$dst"
for f in "$src"/*.go; do
  sed -e "s/__Name__/$Name/g" -e "s/__name__/$name/g" "$f" > "$dst/$(basename "$f")"
done
gofmt -w "$dst"

cat <<EOF
Created $dst. Next steps:
  1. Add it to internal/app/modules.go:
       ${name}s := $name.New($name.Deps{DB: c.DB, Tx: c.Tx})
     and append ${name}s to the returned slice.
  2. make migrate-new name=create_${name}s   (then write the CREATE TABLE)
  3. Document the routes in api/openapi/paths/$name.yaml and schemas/$name.yaml,
     and reference them from api/openapi/openapi.yaml (CI checks coverage).
  4. Add the feature's owners to CODEOWNERS.
EOF
