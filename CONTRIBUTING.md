# Contributing

Thanks for your interest in contributing to DataKit! Your contributions help make this project better for everyone.

By participating, you agree to follow our [Code of Conduct](https://github.com/datakit-dev/.github/blob/main/CODE_OF_CONDUCT.md).

## Getting Started

1. **Fork and Clone:**

   - Fork the `dtkt-sdk` repository.
   - If you'd also like to contribute to other DataKit repositories, additionally fork `dtkt-core`, `dtkt-cli`, and `dtkt-integrations`.
   - Clone your forks locally into the same parent directory:

   ```shell
   # gh repo fork datakit-dev/dtkt-sdk --clone --default-branch-only
   git clone https://github.com/<your-username>/dtkt-sdk.git

   # gh repo fork datakit-dev/dtkt-core --clone --default-branch-only
   git clone https://github.com/<your-username>/dtkt-core.git

   # gh repo fork datakit-dev/dtkt-cli --clone --default-branch-only
   git clone https://github.com/<your-username>/dtkt-cli.git

   # gh repo fork datakit-dev/dtkt-integrations --clone --default-branch-only
   git clone https://github.com/<your-username>/dtkt-integrations.git
   ```

2. **Setup Development Environment:**

   - Ensure you have a compatible [Go](https://golang.org/dl/) version installed (refer to `go.mod` for the required version).
   - Install [Task](https://taskfile.dev/#/installation) for running development tasks and scripts.
   - Navigate into the `dtkt-sdk` directory and run setup to install go modules:

   ```shell
   cd dtkt-sdk
   task setup
   ```

3. **Create a Branch:**

   - Always create a new branch for your contribution:

   ```shell
   git checkout -b feature/my-awesome-feature
   ```

## Making Changes

- Follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification for commit messages.
- Clearly document new features or changes in your commits and Pull Requests (PR).
- Follow this branch naming convention:

  ```
  <type>/<description>
  ```

  - `main`: The main development branch. You should never directly commit to main.
  - `feature/`: For new features (e.g., feature/add-foo-bar).
  - `bugfix/`: For bug fixes (e.g., bugfix/fix-foo-bar).
  - `hotfix/`: For urgent fixes (e.g., hotfix/security-patch-foo-bar).
  - `release/`: For branches preparing a release or backporting bugfixes (e.g., release/v1.1.1). DataKit maintainers will be responsible for such branches.
  - `chore/`: For non-code tasks like dependency, docs updates (e.g., chore/update-docs)

### Sign your work!

The sign-off is a simple line at the end of the explanation for the patch. Your
signature certifies that you wrote the patch or otherwise have the right to pass
it on as an open source patch. The rules are pretty simple: if you can certify
the below (from [developercertificate.org](https://developercertificate.org/)):

```
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.


Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

Then you just add a line to every git commit message:

    Signed-off-by: John Smith <john.smith@email.com>

Use your real name (sorry, no pseudonyms or anonymous contributions.)

If you set your `user.name` and `user.email` git configs, you can sign your
commit automatically with `git commit -s`.

## Linting & Testing

- Ensure existing lints and tests pass before submitting your changes:

```shell
task lint
task test
```

- If you add new features or fix bugs, please include or update relevant tests.

## Pull Requests

- Clearly describe the purpose of your changes in the PR.
- Reference related issues in the PR description (e.g., `Closes #123`).
- Keep the PR focused on a single feature or fix.

### Merge approval

DataKit maintainers use LGTM (Looks Good To Me) in comments on the code review to
indicate acceptance.

A change requires at least 2 LGTMs from the maintainers of each
component affected.

## Code Reviews

- Maintainers will review your PR. Please be responsive to feedback.
- Once approved and passing CI checks, your PR will be merged.

## Reporting Issues

- Search existing issues before filing a new one.
- Include detailed steps to reproduce and expected vs actual behavior.

## Community

Need help or have questions?

- Join our [Discord](https://dtkt.dev/join-discord)
- Start a discussion: [GitHub Discussions](https://github.com/datakit-dev/dtkt-cli/discussions)

## Security

Please report security vulnerabilities responsibly following guidelines in our [SECURITY.md](SECURITY.md).

---

Thanks for contributing! 🎉
