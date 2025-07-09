class Hyperbolic < Formula
  desc "Command-line interface for managing GPU instances on Hyperbolic"
  homepage "https://github.com/HyperbolicLabs/hyperbolic-cli"
  url "https://github.com/HyperbolicLabs/hyperbolic-cli/archive/refs/tags/v0.0.1.tar.gz"
  sha256 "3003635f5fd25fb6216680e3c4c6135a097d05c8ed423c2d3a9bca3602056f36"
  license "MIT"
  head "https://github.com/HyperbolicLabs/hyperbolic-cli.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w", output: bin/"hyperbolic")
  end

  test do
    assert_match "hyperbolic", shell_output("#{bin}/hyperbolic --help")
    
    # Test that the binary exists and is executable
    assert_predicate bin/"hyperbolic", :exist?
    assert_predicate bin/"hyperbolic", :executable?
  end
end 