# Security Policy

You can find more information about DataKit and our security practices at [https://withdatakit.com/security](https://withdatakit.com/security).

## Supported Versions

We currently only support the **latest stable release** of the DataKit SDK. Older versions **do not receive security updates**.

## Reporting a Vulnerability

We take security issues seriously. If you discover a vulnerability in the DataKit SDK, please follow these steps:

### 1. Responsible Disclosure

- Do not publicly disclose security vulnerabilities.
- Report the issue privately to our security team preferably using [Github Advisories](https://github.com/datakit-dev/dtkt-sdk/security) or by email.

  📧 Email: <security@withdatakit.com>

  🔒 GPG Key: [datakit-security.pub.asc](https://withdatakit.com/.well-known/datakit-security.pub.asc)

For more details visit https://withdatakit.com/security/researchers#responsible-disclosure

### 2. What to Include in Your Report

When reporting a vulnerability, please provide:

- A detailed description of the issue.
- Steps to reproduce (if possible).
- The affected version(s).
- Any potential mitigation or patch ideas.

We will acknowledge your report within 48 hours and keep you updated on the progress.

## Security Best Practices

To securely use the DataKit SDK, follow these best practices:

### 1. Use Official Channels

Only download the SDK from official sources like [dtkt.dev](https://dtkt.dev) or our [GitHub Releases](https://github.com/datakit-dev/dtkt-sdk/releases).

### 2. Verify Signed Artifacts

DataKit artifacts are signed using Cosign. Before using a downloaded artifact, verify its integrity:

```sh
cosign verify-blob \
  --key https://dtkt.dev/.well-known/cosign.pub \
  --signature artifact.sig \
  artifact
```

### 3. Check for Vulnerabilities

Regularly scan artifacts for vulnerabilities.

For example:

```sh
grype artifact
# or (more efficiently with our pre-generated sbom)
grype sbom:artifact.sbom.spdx.json
```

### 4. Keep Your Version Updated

Since we currently only support the **latest stable release**, always update the SDK before using it.

## Security Tools Used

We use the following tools to ensure the security of our OSS code repositories, artifacts, and deployments. Some are language dependent, while others are general security tools.

### Secrets & Credential Scanning

- **[GitHub Secret Scanning](https://docs.github.com/en/code-security/secret-scanning)** – Detects and prevents secrets from being committed to repositories.

### Code Quality & Linting

- **[Go Report Card](https://github.com/gojp/goreportcard)** – Generates a report on the quality of a Go project.
- **[golangci-lint](https://github.com/golangci/golangci-lint)** – A fast linters runner for Go, catching security and quality issues early.

### Static & Dynamic Code Analysis

- **[GitHub CodeQL](https://github.com/github/codeql)** – Analyzes source code for security vulnerabilities.
- **[Go Fuzzing](https://go.dev/doc/security/fuzz)** – Helps find security vulnerabilities and bugs by fuzz testing Go code.

### Supply Chain Security & Dependency Management

- **[Dependabot](https://github.com/dependabot)** – Automatically detects and updates vulnerable dependencies in our project.

### SBOM & Vulnerability Scanning

- **[syft](https://github.com/anchore/syft)** – Generates a Software Bill of Materials (SBOM) for tracking dependencies.
- **[grype](https://github.com/anchore/grype)** – Scans images, projects, and SBOM data for known vulnerabilities (CVE detection).

### Code Signing & Integrity

- **[cosign](https://github.com/sigstore/cosign)** – Ensures binary authenticity and integrity by signing artifacts.

### Best Practices

- **[OpenSSF Best Practices](https://www.bestpractices.dev/en)** – Provides a set of best practices for secure software development.
- **[OpenSSF Scorecard](https://github.com/ossf/scorecard)** – Evaluates the security posture of our open source project based on best practices.

### Infrastructure & Kubernetes Security

- **[kubescape](https://github.com/kubescape/kubescape)** – Scans Kubernetes manifests and clusters for misconfigurations and vulnerabilities.
- **[kube-score](https://github.com/zegl/kube-score)** – Analyzes Kubernetes object definitions to ensure best practices.
- **[kubesec](https://github.com/controlplaneio/kubesec)** – Scans Kubernetes manifests and performs security risk analysis.

<!-- ### Runtime Security & Monitoring

- **[Falco](https://github.com/falcosecurity/falco)** – Monitors runtime behavior in a containerized/cloud environment. -->

This security stack helps ensure that our artifacts remains safe, reliable, and compliant with industry best practices.

## Hall of Fame (Responsible Disclosures)

We appreciate the efforts of security researchers who help us improve DataKit. If you’d like to be publicly credited for reporting a vulnerability on our [Hall of Fame](https://withdatakit.com/security/researchers/hall-of-fame), let us know! 🎖️
