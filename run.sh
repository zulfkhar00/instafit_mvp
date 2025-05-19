set -e # Exit early if any commands fail

(
  cd "$(dirname "$0")" # Ensure compile steps are run within the repository directory
  go build -o /tmp/vton_mvp main.go
)

chmod +x /tmp/vton_mvp
exec /tmp/vton_mvp "$@"
