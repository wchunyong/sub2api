// Command quick-import installs gateway configuration without a Python runtime.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
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
		return fail("Usage: quick-import install|clean --agent claude|codex|opencode [options]")
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
		return fail("Choose --agent claude, codex or opencode")
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
		fmt.Fprintf(out, "Restore the most recent Sub2API import for %s only. Other Agents are preserved.\n", *agent)
		if !*yes {
			fmt.Fprint(out, "Continue? [y/N] ")
			answer, _ := bufio.NewReader(in).ReadString('\n')
			if strings.ToLower(strings.TrimSpace(answer)) != "y" {
				return 0
			}
		}
		if native.Clean(root, *agent) != nil {
			return fail("Recovery failed; check permissions and recovery information")
		}
		fmt.Fprintln(out, "Restored. Restart the client. Run again to undo an earlier import, if needed.")
		return 0
	}
	if err = native.Preflight(*agent, !*skip); err != nil {
		return fail(err.Error())
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
		payload, err = network.Exchange(*server, *ticket, *agent)
		if err != nil {
			return fail("Failed during configuration exchange: " + err.Error())
		}
	}
	if err = native.ValidateTransportPayload(payload, *agent); err != nil {
		return fail(err.Error())
	}
	if !*stdin {
		if err = network.SynchronizeModels(&payload); err != nil {
			return fail("Failed during model catalog: " + err.Error())
		}
		if *agent == "claude" && !payload.ClaudeModelPickerSupported {
			fmt.Fprintln(out, "Claude Code < 2.1.242: selected model is configured. Upgrade for the full lianjieai /model menu.")
		}
	}
	if native.Install(root, payload) != nil {
		return fail("Configuration write failed. Original configuration was preserved or recovery information is available; check permissions and configuration syntax.")
	}
	fmt.Fprintf(out, "Configured %s. Restart the client. A project configuration may override user settings.\n", *agent)
	fmt.Fprintf(out, "Offline recovery is saved in %s\n", filepath.Join(root, ".sub2api-quick-import", *agent))
	return 0
}
