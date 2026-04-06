# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability, please open an issue with the `security` label. We will aim to respond within 48 hours.

Do not report security vulnerabilities through public GitHub issues.

## Scope

This project is a CLI tool that reads and processes files. It does not:
- Execute arbitrary code
- Write to any files (read-only operations)
- Make network requests (except for MIME type detection from local files)

However, when using aict with shell access, ensure proper sandboxing in your AI agent environment.