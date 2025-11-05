package tools

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestParseOsRelease(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected *DistroInfo
	}{
		{
			name: "Ubuntu os-release",
			content: `NAME="Ubuntu"
VERSION="20.04.4 LTS (Focal Fossa)"
ID=ubuntu
ID_LIKE=debian
VERSION_ID="20.04"
VERSION_CODENAME=focal
UBUNTU_CODENAME=focal`,
			expected: &DistroInfo{
				ID:      "ubuntu",
				Name:    "Ubuntu",
				Version: "20.04",
				Family:  "debian",
			},
		},
		{
			name: "Debian os-release",
			content: `PRETTY_NAME="Debian GNU/Linux 11 (bullseye)"
NAME="Debian GNU/Linux"
VERSION_ID="11"
VERSION_CODENAME=bullseEE
ID=debian
HOME_URL="https://www.debian.org/"
SUPPORT_URL="https://www.debian.org/support"`,
			expected: &DistroInfo{
				ID:      "debian",
				Name:    "Debian GNU/Linux",
				Version: "11",
				Family:  "debian",
			},
		},
		{
			name: "Alpine os-release",
			content: `NAME="Alpine Linux"
ID=alpine
VERSION_ID=3.16.0
HOME_URL="https://alpinelinux.org/"`,
			expected: &DistroInfo{
				ID:      "alpine",
				Name:    "Alpine Linux",
				Version: "3.16.0",
				Family:  "alpine",
			},
		},
		{
			name: "CentOS os-release",
			content: `NAME="CentOS Linux"
VERSION="7 (Core)"
ID="centos"
ID_LIKE="rhel fedora"
VERSION_ID="7"`,
			expected: &DistroInfo{
				ID:      "centos",
				Name:    "CentOS Linux",
				Version: "7",
				Family:  "rhel",
			},
		},
		{
			name: "带引号的内容",
			content: `NAME="Ubuntu"
VERSION="20.04.4 LTS"
ID=ubuntu
VERSION_ID="20.04"`,
			expected: &DistroInfo{
				ID:      "ubuntu",
				Name:    "Ubuntu",
				Version: "20.04",
				Family:  "debian",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOsRelease(tt.content)

			if result.ID != tt.expected.ID {
				t.Errorf("ID不匹配: 期望 %s，实际 %s", tt.expected.ID, result.ID)
			}
			if result.Name != tt.expected.Name {
				t.Errorf("Name不匹配: 期望 %s，实际 %s", tt.expected.Name, result.Name)
			}
			if result.Version != tt.expected.Version {
				t.Errorf("Version不匹配: 期望 %s，实际 %s", tt.expected.Version, result.Version)
			}
			if result.Family != tt.expected.Family {
				t.Errorf("Family不匹配: 期望 %s，实际 %s", tt.expected.Family, result.Family)
			}
		})
	}
}

func TestParseLsbRelease(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected *DistroInfo
	}{
		{
			name: "Ubuntu lsb-release",
			content: `DISTRIB_ID=Ubuntu
DISTRIB_RELEASE=20.04
DISTRIB_CODENAME=focal
DISTRIB_DESCRIPTION="Ubuntu 20.04.4 LTS"`,
			expected: &DistroInfo{
				ID:      "ubuntu",
				Name:    "",
				Version: "20.04",
			},
		},
		{
			name: "Debian lsb-release",
			content: `DISTRIB_ID=Debian
DISTRIB_RELEASE=11
DISTRIB_CODENAME=bullseye
DISTRIB_DESCRIPTION="Debian GNU/Linux 11 (bullseye)"`,
			expected: &DistroInfo{
				ID:      "debian",
				Name:    "",
				Version: "11",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLsbRelease(tt.content)

			if result.ID != tt.expected.ID {
				t.Errorf("ID不匹配: 期望 %s，实际 %s", tt.expected.ID, result.ID)
			}
			if result.Version != tt.expected.Version {
				t.Errorf("Version不匹配: 期望 %s，实际 %s", tt.expected.Version, result.Version)
			}
		})
	}
}

func TestDetectDistro(t *testing.T) {
	// 这个测试在不同环境下可能会有不同结果
	// 我们主要测试函数不会崩溃
	distro, err := detectDistro()
	if err != nil {
		t.Errorf("检测发行版失败: %v", err)
	}

	if distro == nil {
		t.Error("发行版信息为空")
	}

	t.Logf("检测到发行版: %+v", distro)
}

func TestGetDistroSpecificCommand(t *testing.T) {
	tests := []struct {
		name     string
		distro   *DistroInfo
		command  string
		expected string
	}{
		{
			name: "Ubuntu update upgrade",
			distro: &DistroInfo{
				ID:     "ubuntu",
				Family: "debian",
			},
			command:  "apt update && apt upgrade -y",
			expected: "export DEBIAN_FRONTEND=noninteractive && apt update && apt upgrade -y",
		},
		{
			name: "Ubuntu 通用命令",
			distro: &DistroInfo{
				ID:     "ubuntu",
				Family: "debian",
			},
			command:  "echo hello",
			expected: "echo hello",
		},
		{
			name: "Alpine install 命令",
			distro: &DistroInfo{
				ID:     "alpine",
				Family: "alpine",
			},
			command:  "apk add nginx",
			expected: "command -v apk >/dev/null 2>&1 && apk add nginx || echo 'Alpine包管理器不可用'",
		},
		{
			name: "Alpine 通用命令",
			distro: &DistroInfo{
				ID:     "alpine",
				Family: "alpine",
			},
			command:  "echo hello",
			expected: "echo hello",
		},
		{
			name: "Debian install 命令",
			distro: &DistroInfo{
				ID:     "debian",
				Family: "debian",
			},
			command:  "apt install nginx",
			expected: "apt install nginx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDistroSpecificCommand(tt.distro, tt.command)
			if result != tt.expected {
				t.Errorf("命令不匹配: 期望 %s，实际 %s", tt.expected, result)
			}
		})
	}
}

func TestGetShellCommand(t *testing.T) {
	tests := []struct {
		name        string
		os          string
		command     string
		expectedCmd string
	}{
		{
			name:        "macOS bash",
			os:          "darwin",
			command:     "echo hello",
			expectedCmd: "bash",
		},
		{
			name:        "Linux通用",
			os:          "linux",
			command:     "echo hello",
			expectedCmd: "bash",
		},
		{
			name:        "默认bash",
			os:          "freebsd",
			command:     "echo hello",
			expectedCmd: "bash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 备份原始 GOOS
			originalOS := runtime.GOOS
			defer func() {
				runtime.GOOS = originalOS
			}()

			// 临时设置 GOOS
			runtime.GOOS = tt.os

			cmd, args := getShellCommand(tt.command)

			if cmd != tt.expectedCmd {
				t.Errorf("Shell命令不匹配: 期望 %s，实际 %s", tt.expectedCmd, cmd)
			}

			// Linux和macOS应该使用 -c 参数
			if tt.os == "linux" || tt.os == "darwin" {
				if len(args) != 2 || args[0] != "-c" {
					t.Errorf("参数不正确: 期望 [-c, command]，实际 %v", args)
				}
			}
		})
	}
}

func TestGetShellCommandWithDistro(t *testing.T) {
	// 这个测试假设我们在Linux环境下运行
	if runtime.GOOS != "linux" {
		t.Skip("跳过非Linux环境的测试")
	}

	// 创建临时的 /etc/os-release 文件用于测试
	originalOsRelease := ""
	if content, err := os.ReadFile("/etc/os-release"); err == nil {
		originalOsRelease = string(content)
		defer func() {
			if originalOsRelease != "" {
				os.WriteFile("/etc/os-release", []byte(originalOsRelease), 0644)
			}
		}()
	}

	// 测试 Ubuntu
	testUbuntu := `NAME="Ubuntu"
VERSION="20.04.4 LTS"
ID=ubuntu
VERSION_ID="20.04"`
	os.WriteFile("/etc/os-release", []byte(testUbuntu), 0644)

	cmd, args := getShellCommand("apt update && apt upgrade -y")
	if cmd != "bash" {
		t.Errorf("Ubuntu: 期望 bash，实际 %s", cmd)
	}
	// 检查是否添加了 DEBIAN_FRONTEND
	if !strings.Contains(args[len(args)-1], "DEBIAN_FRONTEND") {
		t.Errorf("Ubuntu: 期望包含 DEBIAN_FRONTEND，实际 %v", args)
	}

	// 测试 Alpine
	testAlpine := `NAME="Alpine Linux"
ID=alpine
VERSION_ID=3.16.0`
	os.WriteFile("/etc/os-release", []byte(testAlpine), 0644)

	cmd, args = getShellCommand("apk add nginx")
	if cmd != "bash" {
		t.Errorf("Alpine: 期望 bash，实际 %s", cmd)
	}
	// 检查是否包含 apk 检查
	if !strings.Contains(args[len(args)-1], "command -v apk") {
		t.Errorf("Alpine: 期望包含 apk 检查，实际 %v", args)
	}

	// 测试 Debian
	testDebian := `NAME="Debian GNU/Linux"
ID=debian
VERSION_ID="11"`
	os.WriteFile("/etc/os-release", []byte(testDebian), 0644)

	cmd, args = getShellCommand("apt update && apt upgrade -y")
	if cmd != "bash" {
		t.Errorf("Debian: 期望 bash，实际 %s", cmd)
	}
	// 检查是否添加了 DEBIAN_FRONTEND
	if !strings.Contains(args[len(args)-1], "DEBIAN_FRONTEND") {
		t.Errorf("Debian: 期望包含 DEBIAN_FRONTEND，实际 %v", args)
	}

	t.Log("所有发行版特定命令测试通过")
}

func TestCleanANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "包含ANSI转义序列",
			input:    "\x1b[31m红色文本\x1b[0m",
			expected: "红色文本",
		},
		{
			name:     "不包含ANSI转义序列",
			input:    "普通文本",
			expected: "普通文本",
		},
		{
			name:     "复杂ANSI序列",
			input:    "\x1b[1;32m绿色粗体\x1b[0m \x1b[4m下划线\x1b[0m",
			expected: "绿色粗体 下划线",
		},
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanANSI(tt.input)
			if result != tt.expected {
				t.Errorf("清理ANSI序列失败: 期望 %q，实际 %q", tt.expected, result)
			}
		})
	}
}

func TestParseRange(t *testing.T) {
	tests := []struct {
		name     string
		rangeStr string
		total    int
		expected []int // [start, end]
		hasError bool
	}{
		{
			name:     "标准范围",
			rangeStr: "1:10",
			total:    15,
			expected: []int{0, 10},
			hasError: false,
		},
		{
			name:     "从第5行开始",
			rangeStr: "5:",
			total:    15,
			expected: []int{4, 15},
			hasError: false,
		},
		{
			name:     "到第5行结束",
			rangeStr: ":5",
			total:    15,
			expected: []int{0, 5},
			hasError: false,
		},
		{
			name:     "整个范围",
			rangeStr: "1:5",
			total:    10,
			expected: []int{0, 5},
			hasError: false,
		},
		{
			name:     "反向范围",
			rangeStr: "10:5",
			total:    15,
			expected: []int{5, 10},
			hasError: false,
		},
		{
			name:     "超过总行数",
			rangeStr: "1:100",
			total:    10,
			expected: []int{0, 10},
			hasError: false,
		},
		{
			name:     "无效格式",
			rangeStr: "invalid",
			total:    10,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := parseRange(tt.rangeStr, tt.total)

			if tt.hasError {
				if err == nil {
					t.Errorf("期望错误但未发生")
				}
				return
			}

			if err != nil {
				t.Errorf("不期望错误: %v", err)
				return
			}

			if start != tt.expected[0] || end != tt.expected[1] {
				t.Errorf("范围解析错误: 期望 [%d, %d]，实际 [%d, %d]",
					tt.expected[0], tt.expected[1], start, end)
			}
		})
	}
}

func TestResolvePath(t *testing.T) {
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取工作目录失败: %v", err)
	}

	tests := []struct {
		name     string
		filePath string
		basePath string
		expected string
		hasError bool
	}{
		{
			name:     "绝对路径",
			filePath: "/tmp/test.txt",
			basePath: "",
			expected: "/tmp/test.txt",
			hasError: false,
		},
		{
			name:     "相对路径",
			filePath: "test.txt",
			basePath: "",
			expected: wd + "/test.txt",
			hasError: false,
		},
		{
			name:     "带基准路径",
			filePath: "test.txt",
			basePath: "/tmp",
			expected: "/tmp/test.txt",
			hasError: false,
		},
		{
			name:     "基准路径中的相对路径",
			filePath: "subdir/test.txt",
			basePath: "/tmp",
			expected: "/tmp/subdir/test.txt",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolvePath(tt.filePath, tt.basePath)

			if tt.hasError {
				if err == nil {
					t.Errorf("期望错误但未发生")
				}
				return
			}

			if err != nil {
				t.Errorf("不期望错误: %v", err)
				return
			}

			// 清理路径以进行比较
			result = filepath.Clean(result)
			expected := filepath.Clean(tt.expected)

			if result != expected {
				t.Errorf("路径解析错误: 期望 %s，实际 %s", expected, result)
			}
		})
	}
}
