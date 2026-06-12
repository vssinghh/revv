# E2E Test Infra: revv

## Test Philosophy
- Opaque-box, requirement-driven. Executes the compiled `bin/revv` binary under various environment states, flag inputs, and repository contexts.
- Methodology: Category-Partition (equivalence partition inputs) + Boundary Value Analysis (missing files, empty context, empty keys) + Pairwise Interaction Testing (model flag + key combo) + Workload Testing (realistic open source repository layouts).

## Feature Inventory
| # | Feature | Source (requirement) | Tier 1 | Tier 2 | Tier 3 |
|---|---------|---------------------|:------:|:------:|:------:|
| 1 | CLI Help & Args Routing | R3. CLI Entrypoint and Help | 5 | 5 | ✓ |
| 2 | Auth & Model Flags | R2. `--model` & `GEMINI_API_KEY` | 5 | 5 | ✓ |
| 3 | Context Gathering | R2. Context files reading | 5 | 5 | ✓ |
| 4 | Scaffold Generation | R2. `.revv/` directory layout | 5 | 5 | ✓ |
| 5 | Git Branch & Commit | R4. Git PR preparation | 5 | 5 | ✓ |

## Test Architecture
- **Location**: `tests/e2e/`
- **Test Runner**: A Go test file `tests/e2e/e2e_test.go` that builds the binary, creates mock repositories, sets environment variables, runs the binary, and checks side effects.
- **Pass/Fail Semantics**: The test runner exits with 0 if all 60 test cases pass, and non-zero otherwise.
- **Mocking**: Sets `REVV_MOCK_LLM=true` to mock LLM configuration generation, ensuring testing correctness is deterministic and fast.

## Test Case Inventory

### Tier 1: Feature Coverage (25 tests, 5 per feature)
- **F1 (CLI Help)**:
  1. `e2e_f1_help_global`: Verify `revv --help` output shows usage and `init` subcommand.
  2. `e2e_f1_help_init`: Verify `revv init --help` output shows `--model` flag and command details.
  3. `e2e_f1_invalid_command`: Verify running `revv invalid` returns usage error and exit code 1.
  4. `e2e_f1_no_args`: Verify running `revv` with no args prints short description/help.
  5. `e2e_f1_help_shorthand`: Verify `revv -h` outputs usage text.
- **F2 (Auth & Model)**:
  6. `e2e_f2_no_api_key`: Verify running `revv init` without `GEMINI_API_KEY` prints clear error and exits with code 1.
  7. `e2e_f2_valid_key_empty_repo`: Verify running `revv init` with `GEMINI_API_KEY` set succeeds (skips missing files).
  8. `e2e_f2_default_model`: Verify default model is used when `--model` is omitted.
  9. `e2e_f2_custom_model`: Verify custom model (e.g., `--model gemini-2.5-pro`) is accepted.
  10. `e2e_f2_invalid_model_flag`: Verify invalid model usage logs/errors correctly.
- **F3 (Context Gathering)**:
  11. `e2e_f3_contributing_md`: Verify context gathers from `CONTRIBUTING.md` when present.
  12. `e2e_f3_readme_md`: Verify context gathers from `README.md` when present.
  13. `e2e_f3_makefile`: Verify context gathers from `Makefile` when present.
  14. `e2e_f3_dockerfile`: Verify context gathers from `Dockerfile` when present.
  15. `e2e_f3_mixed_files`: Verify context gathers from a subset of present files.
- **F4 (Scaffold Generation)**:
  16. `e2e_f4_dockerfile_created`: Verify `.revv/Dockerfile` is created.
  17. `e2e_f4_manual_tests_created`: Verify `.revv/manual/` directory is created.
  18. `e2e_f4_unit_tests_created`: Verify `.revv/unit/` or similar categories are created.
  19. `e2e_f4_test_md_format`: Verify generated `test.md` files contain Description, Priority, Commands, and Expected Output headings.
  20. `e2e_f4_helpers_created`: Verify `.revv/helpers/` or sub-helpers are created.
- **F5 (Git Branch & Commit)**:
  21. `e2e_f5_branch_exists`: Verify branch `revv/init` is created.
  22. `e2e_f5_commit_created`: Verify git commit is created.
  23. `e2e_f5_commit_message`: Verify commit message is descriptive.
  24. `e2e_f5_files_staged`: Verify only `.revv/` directory files are committed.
  25. `e2e_f5_push_instructions`: Verify next-step instructions mentioning `git push` are printed to stdout.

### Tier 2: Boundary & Corner Cases (25 tests)
- **F1 (CLI Help)**:
  26. `e2e_f1_extra_args`: `revv init extra_arg` prints error.
  27. `e2e_f1_unknown_flag`: `revv init --unknown` prints error and exit code 1.
  28. `e2e_f1_duplicate_help`: `revv init --help -h` handles gracefully.
  29. `e2e_f1_help_override`: `revv init --help` overrides missing API key check.
  30. `e2e_f1_empty_init_arg`: `revv init ""` handles gracefully.
- **F2 (Auth & Model)**:
  31. `e2e_f2_empty_api_key`: `GEMINI_API_KEY=""` (explicitly empty string) yields error.
  32. `e2e_f2_spaces_api_key`: `GEMINI_API_KEY="   "` handles gracefully.
  33. `e2e_f2_large_model_name`: `--model` flag with very large name handles gracefully.
  34. `e2e_f2_special_chars_model`: `--model` flag with special chars handles.
  35. `e2e_f2_env_model_override`: Model overrides default in multi-flag combos.
- **F3 (Context Gathering)**:
  36. `e2e_f3_empty_files`: README.md and CONTRIBUTING.md are empty files.
  37. `e2e_f3_massive_files`: README.md size is very large (e.g. 1MB).
  38. `e2e_f3_no_readable_files`: No files exist in repo root.
  39. `e2e_f3_unreadable_files`: Context file permissions set to write-only (unreadable).
  40. `e2e_f3_special_chars_in_files`: Context files contain non-UTF-8 characters.
- **F4 (Scaffold Generation)**:
  41. `e2e_f4_existing_revv_dir`: `.revv/` already exists with conflicting contents.
  42. `e2e_f4_read_only_revv_dir`: `.revv/` is read-only.
  43. `e2e_f4_empty_config_output`: Mock returns empty list of tests (gracefully handles).
  44. `e2e_f4_special_chars_test_names`: Test category/names contain special chars.
  45. `e2e_f4_large_dockerfile`: Dockerfile is very large.
- **F5 (Git Branch & Commit)**:
  46. `e2e_f5_no_git_repo`: Run `revv init` outside a git repository (should return error).
  47. `e2e_f5_dirty_workdir`: Git repository has uncommitted modifications (staged/unstaged).
  48. `e2e_f5_existing_init_branch`: Git branch `revv/init` already exists.
  49. `e2e_f5_untracked_files_outside_revv`: Untracked files exist outside `.revv/` (should not be committed).
  50. `e2e_f5_detached_head`: Run `revv init` in detached HEAD state.

### Tier 3: Cross-Feature Combinations (5 tests)
51. `e2e_t3_key_and_model_combinations`: Combination of `--model` and `GEMINI_API_KEY` options.
52. `e2e_t3_dirty_repo_and_custom_model`: Run on dirty repo with custom model flag.
53. `e2e_t3_no_git_and_no_key`: Run outside git repo without API key (check error precedence).
54. `e2e_t3_existing_branch_and_existing_dir`: Re-run command when both branch and `.revv/` exist.
55. `e2e_t3_empty_context_and_valid_git`: Run with empty context files in valid git repository.

### Tier 4: Real-World Application Scenarios (5 tests)
56. `e2e_t4_standard_go_repo`: A standard Go repository with README, CONTRIBUTING, Makefile, and Dockerfile.
57. `e2e_t4_node_repo`: A Node.js repository with package.json, README.md, and Dockerfile.
58. `e2e_t4_python_repo`: A Python repository with requirements.txt, CONTRIBUTING.md, and Makefile.
59. `e2e_t4_cplusplus_repo`: A C++ repository with CMakeLists.txt and README.md.
60. `e2e_t4_multiple_categories`: Scenario where LLM generates multiple categories (unit, integration, lint, manual).

## Coverage Thresholds
- Tier 1: ≥5 per feature (25 total)
- Tier 2: ≥5 per feature (25 total)
- Tier 3: pairwise coverage of major features (5 total)
- Tier 4: 5 realistic application-level scenarios
- Total: 60 test cases
