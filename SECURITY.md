# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability, please open an issue with the `security` label. We will aim to respond within 48 hours.

Do not report security vulnerabilities through public GitHub issues.

## Note on AI-Generated Code

This project was built entirely by AI tools. While we strive for secure code, AI-generated software may contain unexpected vulnerabilities. We recommend:
- Audit the code before using in production
- Report any issues you find so they can be addressed
- Use proper sandboxing regardless

## Scope

This project is a CLI tool that reads and processes files. It does not:
- Execute arbitrary code
- Write to any files (read-only operations)
- Make network requests (except for MIME type detection from local files)

However, when using aict with shell access, ensure proper sandboxing in your AI agent environment.