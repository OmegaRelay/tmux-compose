package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type Pane struct {
	Object  `yaml:",inline"`
	Dir     string
	Focus   bool
	Cmd     string
	KillCmd string `yaml:"kill_cmd"`
	target  string
}

type Window struct {
	Object `yaml:",inline"`
	Dir    string
	Focus  bool
	Layout string
	Panes  []*Pane
}

type Session struct {
	Object  `yaml:",inline"`
	Dir     string
	Windows []*Window
	started bool
}

type Project struct {
	Dir         string
	UpPreCmd    string `yaml:"up_pre_cmd"`
	UpPostCmd   string `yaml:"up_post_cmd"`
	DownPreCmd  string `yaml:"down_pre_cmd"`
	DownPostCmd string `yaml:"down_post_cmd"`
	Server      string
	Sessions    []*Session
}

var gShellArgs []string
var gRestart bool
var gTmuxArgs string

func shellRun(format string, args ...interface{}) error {
	cmdStr := fmt.Sprintf(format, args...)
	cmd := exec.Command(gShellArgs[0], append(gShellArgs[1:], cmdStr)...)

	fmt.Println(cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s (%w)", out, err)
	}

	return nil
}

func shell(format string, args ...interface{}) {
	err := shellRun(format, args...)
	if err != nil {
		log.Fatal(err)
	}
}

func shellInDir(dir, cmd string) {
	shell("cd \"%s\";%s", coalesce(dir, "."), cmd)
}

func NewWindow(session *Session, window *Window, dir string) {
	var namedWindow string
	if len(window.Name) > 0 {
		namedWindow = fmt.Sprintf(`-n "%s"`, window.Name)
	}
	if session.started {
		shell("tmux %s new-window -d -t \"%s\" \"%s\" -c %s", gTmuxArgs, session.Name, namedWindow, dir)
	} else {
		shell("tmux %s new-session -d -s \"%s\" \"%s\" -c %s", gTmuxArgs, session.Name, namedWindow, dir)
		session.started = true
	}
}

func NewPane(target, dir string) {
	shell("tmux %s split-window -t \"%s\" -c %s", gTmuxArgs, target, dir)
}

func SelectWindow(target string) {
	shell("tmux %s select-window -t \"%s\"", gTmuxArgs, target)
}

func SelectLayout(target, layout string) {
	if layout == "" {
		return
	}
	switch layout {
	case "even-horizontal":
	case "even-vertical":
	case "main-horizontal":
	case "main-vertical":
	case "titled":
	default:
		log.Fatal("Bad layout: " + layout)
	}
	shell("tmux %s select-layout -t \"%s\" \"%s\"", gTmuxArgs, target, layout)
}

func SendLine(target, text string) {
	if text == "" {
		return
	}
	shell("tmux %s send-keys -t \"%s\" '%s'", gTmuxArgs, target, text)
	shell("tmux %s send-keys -R -t \"%s\" 'Enter'", gTmuxArgs, target)
}

func KillSession(session string) {
	shellRun("tmux %s kill-session -t \"%s\"", gTmuxArgs, session)
}

func SetEnvironment(session, key, value string) {
	shell("tmux %s set-environment -t \"%s\" \"%s\" \"%s\"", gTmuxArgs, session, key, value)
}

func coalesce(args ...string) string {
	for _, s := range args {
		if s != "" {
			return s
		}
	}
	return ""
}

func (project *Project) getDir(s *Session, w *Window, paneIndex int) string {
	if paneIndex > len(w.Panes) {
		log.Fatal("Pane index out of bounds?!")
	}

	if paneIndex == 0 && len(w.Panes) == 0 {
		// The window has no explicit panes
		return coalesce(w.Dir, s.Dir, project.Dir, ".")
	}

	return coalesce(w.Panes[paneIndex].Dir, w.Dir, s.Dir, project.Dir, ".")
}

func (p *Pane) Run() {
	if gRestart && p.KillCmd == "" {
		return
	}
	SendLine(p.target, p.Cmd)
}

func (w *Window) Run() {
}

func (s *Session) Run() {
}

func (w *Window) DoReadyCheck() {
	for {
		ready := true

		for _, p := range w.Panes {
			if p == nil {
				continue
			}
			if !p.IsReady() {
				ready = false
				time.Sleep(100 * time.Millisecond)
				break
			}
		}

		if ready {
			return
		}
	}
}

func (s *Session) DoReadyCheck() {
	for {
		ready := true

		for _, w := range s.Windows {
			if w == nil {
				continue
			}
			if !w.IsReady() {
				ready = false
				time.Sleep(100 * time.Millisecond)
				break
			}
		}

		if ready {
			return
		}
	}
}

func (p *Pane) DoReadyCheck() {
	if p.ReadyCheck.Test == "" {
		return
	}

	for {
		if err := shellRun(p.ReadyCheck.Test); err == nil {
			break
		}

		if p.ReadyCheck.Retries <= 0 {
			log.Fatal("Object test failed?!")
		} else {
			p.ReadyCheck.Retries--
			time.Sleep(p.ReadyCheck.Interval)
		}
	}
}

func (project *Project) up() {
	if project.UpPreCmd != "" {
		shellInDir(project.Dir, project.UpPreCmd)
	}

	// Spawn all the sessions/windows/panes
	for _, s := range project.Sessions {
		for wi, w := range s.Windows {
			if w == nil {
				continue
			}
			target := fmt.Sprintf("%s:+%d", s.Name, wi)
			dir := project.getDir(s, w, 0)

			NewWindow(s, w, dir)

			for pi, p := range w.Panes {
				if p == nil {
					continue
				}
				p.target = fmt.Sprintf("%s:+%d.+%d", s.Name, wi, pi)
				dir := project.getDir(s, w, pi)
				if pi > 0 {
					NewPane(target, dir)
				}
			}

			SelectLayout(target, w.Layout)
		}
	}

	// Set which window has focus
	for _, s := range project.Sessions {
		for wi, w := range s.Windows {
			if w == nil {
				continue
			}
			if w.Focus {
				target := fmt.Sprintf("%s:+%d", s.Name, wi)
				SelectWindow(target)
			}
		}
	}

	// Run the commands concurrently
	for _, s := range project.Sessions {
		if s == nil {
			continue
		}
		log.Printf("adding session runner %s", s.Path)
		addRunner(s)
		for _, w := range s.Windows {
			if w == nil {
				continue
			}
			log.Printf("adding window runner %s", w.Path)
			addRunner(w)
			for _, p := range w.Panes {
				if p == nil {
					continue
				}
				log.Printf("adding pane runner %s", p.Path)
				addRunner(p)
			}
		}
	}
	runAll()

	if project.UpPostCmd != "" {
		shellInDir(project.Dir, project.UpPostCmd)
	}
}

func (project *Project) down() {
	if project.DownPreCmd != "" {
		shellInDir(project.Dir, project.DownPreCmd)
	}

	for _, s := range project.Sessions {
		KillSession(s.Name)
	}

	if project.DownPostCmd != "" {
		shellInDir(project.Dir, project.DownPostCmd)
	}
}

func (project *Project) restart() {
	// Run the commands concurrently
	for _, s := range project.Sessions {
		for wi, w := range s.Windows {
			if w == nil {
				continue
			}
			for pi, p := range w.Panes {
				if p == nil {
					continue
				}
				p.target = fmt.Sprintf("%s:+%d.+%d", s.Name, wi, pi)

				if p.KillCmd != "" {
					SendLine(p.target, p.KillCmd)
				}
			}
		}
	}

	// Used in each Panel's Run() method to know if we are performing a restart.
	// Only commands with a KillCmd will be restarted.
	gRestart = true

	// Run the commands concurrently
	for _, s := range project.Sessions {
		if s == nil {
			continue
		}
		addRunner(s)
		for _, w := range s.Windows {
			if w == nil {
				continue
			}
			addRunner(w)
			for _, p := range w.Panes {
				if p == nil {
					continue
				}
				addRunner(p)
			}
		}
	}
	runAll()
}

func (project *Project) attach() {
	args := []string{}
	if project.Server != "" {
		args = append(args, "-L", project.Server)
	}
	args = append(args, "attach")

	binary, err := exec.LookPath("tmux")
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command(binary, args...)
	log.Printf("%s", strings.Join(cmd.Args, " "))
	err = syscall.Exec(binary, cmd.Args, os.Environ())
	if err != nil {
		log.Fatalf("could not exec command: %v", err)
	}
}

func (p *Project) genPaths() {
	for si, s := range p.Sessions {
		s.Path = strconv.FormatInt(int64(si), 10)
		for wi, w := range s.Windows {
			w.Path = filepath.Join(s.Path, strconv.FormatInt(int64(wi), 10))
			for pi, p := range w.Panes {
				p.Path = filepath.Join(w.Path, strconv.FormatInt(int64(pi), 10))
			}
		}
	}
}

func initCmd(cmd *cobra.Command, _ []string) *Project {
	shellArgs := cmd.Flag("shell").Value.String()
	composeFile := cmd.Flag("file").Value.String()

	fmt.Printf("Using shell args: %s\n", shellArgs)
	gShellArgs = strings.Split(shellArgs, " ")

	data, err := os.ReadFile(composeFile)
	if err != nil {
		log.Fatal(err)
	}

	var project Project

	err = yaml.UnmarshalStrict(data, &project)
	if err != nil {
		log.Fatal(err)
	}

	if project.Server != "" {
		gTmuxArgs = fmt.Sprintf("-L \"%s\"", project.Server)
	}

	project.genPaths()
	return &project
}
