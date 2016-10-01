package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
)

type Cmd struct {
	cowerPath   string
	makePkgPath string
}

func findPgm(pgm string) (p string, err error) {
	p, err = exec.LookPath(pgm)
	if err != nil {
		return "", errors.Wrapf(err, "Unable to find %s command.", pgm)
	}
	if !path.IsAbs(p) {
		curDir, err := os.Getwd()
		if err != nil {
			return "", errors.Wrapf(err, "Unable to determine current working directory, so we cannot build path to %s command", pgm)
		}
		p = path.Join(curDir, p)
	}
	return
}

// Find finds the path to the cower command.
func Find() (*Cmd, error) {
	cowerPath, err := findPgm("cower")
	if err != nil {
		return nil, errors.Wrap(err, "Look for cower here: https://aur.archlinux.org/packages/cower/")
	}

	makePkgPath, err := findPgm("makepkg")
	if err != nil {
		return nil, err
	}

	return &Cmd{cowerPath, makePkgPath}, nil
}

func (c *Cmd) QueryUpdates() ([]Update, error) {
	cmd := exec.Command(c.cowerPath, "--update")
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "Getting stdout for cower --update")
	}
	if err = cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "starting cower --update")
	}
	pkgs := []Update{}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		// Example line:
		// :: google-chrome 53.0.2785.116-1 -> 53.0.2785.143-1
		var colons, name, curver, arrow, newver string
		_, err = fmt.Sscan(line, &colons, &name, &curver, &arrow, &newver)
		if err != nil {
			return nil, errors.Wrapf(err, "Unable to parse [%s]", line)
		}
		pkgs = append(pkgs, Update{name, curver, newver})
	}
	err = cmd.Wait()
	// an exit code of 1 is normal if cower found something to update
	if status, ok := exitStatus(err); !ok || (status != 0 && status != 1) {
		return nil, errors.Wrap(err, "executing cower --update")
	}
	return pkgs, nil
}

func (c *Cmd) Fetch(dir string, logDir string, name string) error {
	cmd := exec.Command(c.cowerPath, "--download", name)
	cmd.Dir = dir
	stdout, err := os.Create(path.Join(logDir, "fetch.out"))
	if err != nil {
		return errors.Wrapf(err, "Fetching %s", name)
	}
	defer stdout.Close()
	cmd.Stdout = stdout

	stderr, err := os.Create(path.Join(logDir, "fetch.err"))
	if err != nil {
		return errors.Wrapf(err, "Fetching %s", name)
	}
	defer stderr.Close()
	cmd.Stderr = stderr

	if err = cmd.Run(); err != nil {
		return errors.Wrapf(err, "running cower --download %s.  See %s", name, logDir)
	}
	return nil
}

func (c *Cmd) Make(dir string, logDir string, name string, skippgpcheck bool) (pkgPath string, err error) {
	cmd := exec.Command(c.makePkgPath, "--syncdeps", name)
	if skippgpcheck {
		cmd.Args = append(cmd.Args, "--skippgpcheck")
	}
	cmd.Dir = path.Join(dir, name)
	stdout, err := os.Create(path.Join(logDir, "make.out"))
	if err != nil {
		return "", errors.Wrapf(err, "Making %s", name)
	}
	defer stdout.Close()
	cmd.Stdout = stdout

	stderr, err := os.Create(path.Join(logDir, "make.err"))
	if err != nil {
		return "", errors.Wrapf(err, "Making %s", name)
	}
	defer stderr.Close()
	cmd.Stderr = stderr

	if err = cmd.Run(); err != nil {
		return "", errors.Wrapf(err, "running makepkg for %s.  See %s", name, logDir)
	}

	matches, err := filepath.Glob(path.Join(cmd.Dir, "*.pkg.*"))
	if err != nil {
		return "", errors.Wrapf(err, "globbing makepkg for %s.  See %s", name, cmd.Dir)
	}
	if len(matches) != 1 {
		return "", errors.Errorf("Expected exactly 1 match but got this instead: %v", matches)
	}

	return matches[0], nil
}

// Update represents what we know about a package that can be updated
type Update struct {
	// Name is the name of this package (e.g. 'google-chrome')
	Name string
	// CurrentVersion is the version of this package currently installed.
	CurrentVersion string
	// NextVersion is the available version of this package
	NextVersion string
}

func (u Update) String() string {
	return fmt.Sprintf(":: %s %s -> %s", u.Name, u.CurrentVersion, u.NextVersion)
}

// see http://stackoverflow.com/questions/10385551/get-exit-code-go
func exitStatus(err error) (status int, ok bool) {
	if err == nil {
		return 0, true
	}

	exiterr, ok := err.(*exec.ExitError)
	if !ok {
		return 0, false
	}
	// The program has exited with an exit code != 0

	// This works on both Unix and Windows. Although package
	// syscall is generally platform dependent, WaitStatus is
	// defined for both Unix and Windows and in both cases has
	// an ExitStatus() method with the same signature.
	waitStatus, ok := exiterr.Sys().(syscall.WaitStatus)
	if !ok {
		return 0, false
	}
	return waitStatus.ExitStatus(), true
}
