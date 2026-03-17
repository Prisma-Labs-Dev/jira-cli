class Jira < Formula
  desc "Agent-first Jira CLI"
  homepage "https://github.com/Prisma-Labs-Dev/jira-cli"
  license "MIT"

  head "file://#{Pathname(__dir__).parent.realpath}", branch: "main", using: GitDownloadStrategy

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(output: bin/"jira"), "."
  end

  test do
    assert_match "Agent-first Jira CLI", shell_output("#{bin}/jira --help")
    assert_match "issue search", shell_output("#{bin}/jira issue search --help")
  end
end
