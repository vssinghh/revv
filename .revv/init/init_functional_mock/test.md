## Description
Execute end-to-end repository initialization under mock LLM configurations and verify that files are created in the target .revv folder and automatically committed to a newly created branch 'revv/init'.

## Priority
blocking

## Commands
```bash
# Source local assertion helpers
. .revv/helpers/assert.sh

# Initialize isolated dummy repository
tmpdir="/tmp/test_init_repo"
rm -rf "$tmpdir"
mkdir -p "$tmpdir"
cd "$tmpdir"

git init
git config user.name "Automated Reviewer"
git config user.email "reviewer@example.com"
touch README.md
git add README.md
git commit -m "Initial commit"

# Execute revv init pointing to compiled bin
export GEMINI_API_KEY="mock-key"
export REVV_MOCK_LLM="true"
/workspace/bin/revv init --verbose

# Verify physical output structural completeness
assert_file_exists ".revv/Dockerfile"
assert_dir_exists ".revv/visual"
assert_dir_exists ".revv/helpers"

# Verify git branch creation and commit tracking
git rev-parse --verify revv/init || (echo "FAIL: Branch revv/init not created" && exit 1)

last_commit_msg=$(git log -n 1 --format=%s)
if [ "$last_commit_msg" != "Initialize revv configuration" ]; then
  echo "FAIL: Unexpected commit message: $last_commit_msg"
  exit 1
fi

echo "PASS: Mock initialization successfully created and committed review config"
```

## Expected Output
Exit code 0. Final line: "PASS: Mock initialization successfully created and committed review config".
