class Hyperbolic < Formula
  desc "Command-line interface for managing GPU instances on Hyperbolic"
  homepage "https://github.com/HyperbolicLabs/hyperbolic-cli"
  url "https://github.com/HyperbolicLabs/hyperbolic-cli/archive/refs/tags/v0.0.3.tar.gz"
  sha256 "c6c88cbd0f48d67df9688a50b59b13fc3bd6b24cd4274e142f63d7fd1d8b7928"
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