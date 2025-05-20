set -e # Exit early if any commands fail

(
  cd "$(dirname "$0")" # Ensure compile steps are run within the repository directory
  go build -o /tmp/instafit_mvp main.go
)

chmod +x /tmp/instafit_mvp
export APP_ENV=dev

# Run Python FastAPI server in the background
(
  cd internal/image_segmentator
  uvicorn app:app --host 127.0.0.1 --port 8000 &
)
# Wait for the Python server to be ready
sleep 5

# Run the Go backend
exec /tmp/instafit_mvp "$@"
