# typed: false
# frozen_string_literal: true

class Aict < Formula
  desc "CLI tools with XML/JSON output for AI agents"
  homepage "https://github.com/synseqack/aict"
  version "1.0.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/synseqack/aict/releases/download/v#{version}/aict-v#{version}-darwin-arm64.tar.gz"
      sha256 "PLACEHOLDER"
    elsif Hardware::CPU.intel?
      url "https://github.com/synseqack/aict/releases/download/v#{version}/aict-v#{version}-darwin-amd64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/synseqack/aict/releases/download/v#{version}/aict-v#{version}-linux-arm64.tar.gz"
      sha256 "PLACEHOLDER"
    elsif Hardware::CPU.intel?
      url "https://github.com/synseqack/aict/releases/download/v#{version}/aict-v#{version}-linux-amd64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  def install
    bin.install "aict"
    bin.install "aict-mcp"
    bash_completion.install "completions/aict.bash" => "aict"
    zsh_completion.install "completions/aict.zsh" => "_aict"
  end

  test do
    assert_match "aict", shell_output("#{bin}/aict --help")
  end
end
