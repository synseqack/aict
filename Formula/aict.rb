# typed: false
# frozen_string_literal: true

class Aict < Formula
  desc "Unix coreutils with XML/JSON output, built for AI agents"
  homepage "https://github.com/synseqack/aict"
  version "2.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/synseqack/aict/releases/download/v#{version}/aict-v#{version}-darwin-arm64.tar.gz"
      sha256 "8e1a04d445988e6d543c1a5eb8fea975f3946fdfa6fdf023237b222be6c95ce4"
    elsif Hardware::CPU.intel?
      url "https://github.com/synseqack/aict/releases/download/v#{version}/aict-v#{version}-darwin-amd64.tar.gz"
      sha256 "2ff0ab27af6cdee9d8b8f15e5336380cc83ba5276d36bffbb76da072e6c895ed"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/synseqack/aict/releases/download/v#{version}/aict-v#{version}-linux-arm64.tar.gz"
      sha256 "a52d7f3961659210ad3f4e0304fa9c814f551aebf1808037e40ff80cfd06f9b0"
    elsif Hardware::CPU.intel?
      url "https://github.com/synseqack/aict/releases/download/v#{version}/aict-v#{version}-linux-amd64.tar.gz"
      sha256 "4db1523bb78b316804fc3b520578da2b710abfa6e5be319c97c50df9bfe5e92f"
    end
  end

  def install
    bin.install "aict"
    bash_completion.install "completions/aict.bash" => "aict"
    zsh_completion.install "completions/aict.zsh" => "_aict"
  end

  test do
    assert_match "aict v#{version}", shell_output("#{bin}/aict --version")
  end
end
