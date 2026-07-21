# Homebrew Formula —— envonce（源码构建版）
#
# 使用说明:
#   本文件随主仓库提交，作为 homebrew-tap 仓库 Formula/envonce.rb 的模板。
#   envonce 是纯 Go 项目（CGO_ENABLED=0），源码构建可通吃所有架构、
#   发布流程最简（只需打 tag），且与 homebrew-core 收录要求一致。
#
# 发布流程（每次新版本）:
#   1. 在主仓库打 tag（如 v1.0.0）并推送。
#   2. 计算 tag 源码包的 sha256:
#        curl -sL https://github.com/laidbackgeek/envonce/archive/refs/tags/v1.0.0.tar.gz | shasum -a 256
#   3. 用得到的 sha 替换下方 SOURCE_SHA256；同步更新 url 中的 tag 与 version 字段。
#   4. 复制本文件到 homebrew-tap 仓库的 Formula/envonce.rb 并推送。
#
# 用户侧安装: brew tap laidbackgeek/homebrew-tap && brew install envonce
# 用户侧升级: 作者推新 tag + 更新本文件 sha 后，用户 brew upgrade envonce 自动从源码重建。

class Envonce < Formula
  desc "统一管理 shell 与 launchd 服务的环境变量"
  homepage "https://github.com/laidbackgeek/envonce"
  url "https://github.com/laidbackgeek/envonce/archive/refs/tags/v1.0.1.tar.gz"
  sha256 "b4cd3c1d1b72746d04c48c898fdc151da8d3bb7ed318015fa3e414ea367b5a44"
  version "1.0.1"
  license "MIT"
  head "https://github.com/laidbackgeek/envonce.git", branch: "main"

  livecheck do
    url :stable
    strategy :github_latest
  end

  depends_on "go" => :build

  def install
    # 注入版本号到 main.version（缺省 "dev"）；CGO_ENABLED=0 由 std_go_args 保证。
    ldflags = "-s -w -X main.version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/envonce"
  end

  def caveats
    <<~EOS
      请运行 envonce init 完成初始化（配置 shell 集成）。

        envonce init

      初始化后重启终端或 source 对应 rc 文件即可启用 envonce 管理的环境变量。
    EOS
  end

  test do
    assert_match "envonce", shell_output("#{bin}/envonce --help")
  end
end
