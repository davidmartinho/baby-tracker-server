# Instructions for Codex Cloud Agents

This file contains instructions for Codex Cloud agents working on this repository.

## Automatic Pull Request Creation

**IMPORTANT**: Whenever you complete a task or make code changes, you MUST automatically create a Pull Request (PR) on GitHub.

### Mandatory Process

1. **After completing any task**:
   - Commit all changes
   - Create a branch with a descriptive name (e.g., `feature/add-user-authentication`, `fix/api-token-validation`)
   - Automatically open a Pull Request

2. **Pull Request Format**:
   - **Title**: Clearly describe what was done (e.g., "Add user authentication via API token")
   - **Description**: Include:
     - Summary of changes
     - Motivation/context
     - List of main changes
     - Any relevant information for review

3. **Never commit directly to the `main` branch**:
   - Always work on a separate branch
   - Always open a PR for review before merging

### Workflow Example

```bash
# 1. Create branch
git checkout -b feature/feature-name

# 2. Make changes and commits
git add .
git commit -m "Clear description of changes"

# 3. Push branch
git push origin feature/feature-name

# 4. Automatically create PR using GitHub CLI or API
gh pr create --title "Descriptive title" --body "Detailed description"
```

### Exceptions

The only exception for not creating a PR is when you are only:
- Answering questions or providing information
- Making temporary changes for debug/testing that will be immediately reverted

For any permanent code changes, a PR is mandatory.

---

**Note**: This is a mandatory policy for this repository. Codex Cloud agents must follow this process without exception.
