# revv

`revv` is an LLM-powered Pull Request (PR) review automation tool designed for open-source maintainers. It automatically onboards repositories, gathers context, and uses large language models (LLMs) to generate targeted testing sandboxes and validation criteria.

---

## Features

- **CLI-driven Onboarding (`revv init`)**: Easily initialize testing context within any repository.
- **Context Collection**: Automatically gathers repository files (such as `README.md`, `CONTRIBUTING.md`, `Makefile`, `Dockerfile`, etc.) to understand the project architecture.
- **LLM-Powered Configuration Generation**: Communicates with Gemini via the model-agnostic `adk-go` SDK to generate tailor-made sandbox configurations.
- **Structured Sandbox Output**:
  - Generates `.revv/Dockerfile` optimized for building and running test sandboxes.
  - Groups tests into clean, modular categories (e.g., unit, integration).
  - Automatically generates a `manual` category (`.revv/manual/`) for tests that cannot be automated (UI, UX flows, visual verification).
  - Emits descriptive `test.md` files for every test suite mapping out the test criteria, commands, validation steps, and helper scripts.
- **Automated Git Workflows**: Safely checkpoints the generated configuration in a new branch `revv/init` and prepares a descriptive local commit, ready for push and PR review.

---

## Project Layout

The repository follows standard Go project structure guidelines:

```text
.
├── Makefile                     # Build, test, and clean orchestration
├── README.md                    # Project documentation
├── LICENSE                      # MIT License
├── cmd/
│   └── revv/
│       └── main.go              # CLI command entrypoint
└── internal/
    ├── cli/                     # CLI parsing & command routing (Cobra)
    ├── context/                 # Repository scanner & context collection
    ├── llm/                     # Gemini agent client and prompt structures
    └── git/                     # Git branch & commit automation logic
```

---

## Installation & Build

### Prerequisites
- **Go**: Version 1.26.4 or higher is recommended.
- **Git**: Installed and configured locally.
- **Gemini API Key**: A valid key is required for LLM operations. You can obtain one from [Google AI Studio](https://aistudio.google.com/app/api-keys).

### Building from Source
Run the Makefile build target to compile the CLI binary:
```bash
make build
```
This produces the compiled executable `bin/revv` (or `./revv`).

### Running Tests
To run unit and integration tests:
```bash
make test
```

---

## Usage

### 1. Initialize Configuration
To onboard a repository and generate its PR review sandbox configurations, navigate to the target repository root and run:
```bash
export GEMINI_API_KEY="your-api-key-here"
revv init --model gemini
```

### 2. Flag Options
* `--model <model>`: Specify the LLM model identifier (defaults to `gemini`).
* `-h, --help`: Display detailed usage instructions.

### 3. Review Generated Configurations
The `init` command creates a local `.revv/` directory with the following structure:
```text
.revv/
├── Dockerfile                   # Build environment sandbox
├── helpers/                     # Global shared helpers
│   └── common.sh
├── unit/                        # Categorized automated test suites
│   ├── config_validation/
│   │   └── test.md
│   └── helpers/
│       └── test_fixtures.sh
└── manual/                      # Mandatory manual test suites
    └── visual_checks/
        └── test.md
```

### 4. Push and Open a PR
After generation, `revv` checks out a new branch `revv/init` and commits `.revv/` automatically. Output instructions will guide you to push:
```bash
git push origin revv/init
```
From there, open a PR to merge the review sandbox configuration.

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
