// Command quick-import installs gateway configuration without a Python runtime.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/quickimport/native"
)

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

func run(args []string, in io.Reader, out, stderr io.Writer) int {
	fail := func(message string) int { fmt.Fprintln(stderr, message); return 1 }
	if len(args) == 0 || (args[0] != "install" && args[0] != "clean") {
		return fail("Usage: quick-import install|clean --agent claude|codex|opencode|workbuddy|pi|deepseek-harness [options]")
	}
	flags := flag.NewFlagSet("quick-import", flag.ContinueOnError)
	// Flag parse errors can quote arguments, including an accidentally misplaced key.
	flags.SetOutput(io.Discard)
	agent := flags.String("agent", "", "client name")
	server := flags.String("server", "", "gateway HTTPS URL")
	ticket := flags.String("ticket", "", "one-time import ticket")
	stdin := flags.Bool("stdin", false, "read configuration from stdin")
	home := flags.String("home", "", "isolated home directory")
	skip := flags.Bool("skip-client-check", false, "isolated stdin tests only")
	yes := flags.Bool("yes", false, "confirm recovery")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
		return fail("Invalid command options")
	}
	if !native.ValidAgent(*agent) {
		return fail("Choose a supported --agent value")
	}
	if *skip && (!*stdin || *home == "" || args[0] != "install") {
		return fail("--skip-client-check requires install --stdin and --home")
	}
	if *home == "" {
		var err error
		*home, err = os.UserHomeDir()
		if err != nil {
			return fail("Could not locate home directory")
		}
	}
	root, err := filepath.Abs(*home)
	if err != nil {
		return fail("Invalid home directory")
	}
	if args[0] == "clean" {
		fmt.Fprintf(out, "正在恢复 %s 最近一次连接AI导入；其他工具的配置不会更改。\n", *agent)
		if !*yes {
			fmt.Fprint(out, "确认恢复？输入 y 继续 [y/N] ")
			answer, _ := bufio.NewReader(in).ReadString('\n')
			if strings.ToLower(strings.TrimSpace(answer)) != "y" {
				return 0
			}
		}
		if err := native.Clean(root, *agent); err != nil {
			var conflict *native.RecoveryConflict
			if errors.As(err, &conflict) {
				return fail("原配置在导入后有变化，已保留当前文件：" + conflict.Error())
			}
			return fail("恢复失败。请检查配置文件权限和本机恢复信息。")
		}
		fmt.Fprintln(out, "恢复完成。请完全退出并重启工具；若曾多次导入，可再次运行以恢复更早的版本。")
		return 0
	}
	fmt.Fprintf(out, "正在检查 %s 安装状态和本机配置…\n", *agent)
	if err = native.Preflight(*agent, !*skip); err != nil {
		return fail("安装或配置检查未通过：" + err.Error())
	}
	if err = native.CheckLocalConfig(root, *agent); err != nil {
		return fail("现有本机配置需要先处理；尚未兑换配置码：" + err.Error())
	}
	var payload native.Payload
	network := native.Network{}
	if *stdin {
		data, e := io.ReadAll(io.LimitReader(in, 8*1024*1024+1))
		if e != nil || len(data) > 8*1024*1024 {
			return fail("Invalid input configuration")
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		if decoder.Decode(&payload) != nil || decoder.Decode(new(any)) != io.EOF {
			return fail("Invalid input configuration")
		}
	} else {
		if *server == "" || *ticket == "" {
			return fail("Missing --server or --ticket")
		}
		fmt.Fprintln(out, "正在兑换一次性配置码…")
		payload, err = network.Exchange(*server, *ticket, *agent)
		if err != nil {
			return fail("配置码兑换失败。请返回密钥列表重新生成导入命令。详情：" + err.Error())
		}
	}
	if err = native.ValidateTransportPayload(payload, *agent); err != nil {
		return fail(err.Error())
	}
	if !*stdin {
		fmt.Fprintln(out, "正在读取这把 Key 可用的模型并核对所选模型…")
		if err = network.SynchronizeModels(&payload); err != nil {
			return fail("模型目录读取失败。请检查 Key、分组和网络，再重新生成命令。详情：" + err.Error())
		}
		if *agent == "claude" && !payload.ClaudeModelPickerSupported {
			fmt.Fprintln(out, "Claude Code < 2.1.242: selected model is configured. Upgrade for the full lianjieai /model menu.")
		}
	}
	if native.Install(root, payload) != nil {
		return fail("配置写入失败。原配置已保留，或可使用本机恢复信息撤销；请检查文件权限与配置语法。")
	}
	fmt.Fprintf(out, "%s 配置已写入。请完全退出并重启工具，选择导入时选定的模型，发送一条测试消息。\n", *agent)
	fmt.Fprintln(out, "如果工具仍使用旧服务，请检查项目级配置是否覆盖了用户配置。")
	fmt.Fprintf(out, "本机恢复信息：%s\n", filepath.Join(root, ".sub2api-quick-import", *agent))
	return 0
}
