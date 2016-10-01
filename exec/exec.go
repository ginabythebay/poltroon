package exec

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"

	"github.com/ginabythebay/poltroon"
	"github.com/pkg/errors"
)

type Exec struct {
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
func Find() (*Exec, error) {
	cowerPath, err := findPgm("cower")
	if err != nil {
		return nil, errors.Wrap(err, "Look for cower here: https://aur.archlinux.org/packages/cower/")
	}

	makePkgPath, err := findPgm("makepkg")
	if err != nil {
		return nil, err
	}

	return &Exec{cowerPath, makePkgPath}, nil
}

// QueryUpdates looks for already-installed but out-of-date AUR
// packages, using the cower command.
func (e *Exec) QueryUpdates(pkgsRoot string) ([]*poltroon.AurPackage, error) {
	cmd := exec.Command(e.cowerPath, "--update")
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "Getting stdout for cower --update")
	}
	if err = cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "starting cower --update")
	}
	pkgs := []*poltroon.AurPackage{}
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
		pkgs = append(pkgs, poltroon.NewAurPackage(pkgsRoot, name, curver, newver))
	}
	err = cmd.Wait()
	// an exit code of 1 is normal if cower found something to update
	if status, ok := exitStatus(err); !ok || (status != 0 && status != 1) {
		return nil, errors.Wrap(err, "executing cower --update")
	}
	return pkgs, nil
}

// Fetch fetches a package, using the cower command.
func (e *Exec) Fetch(dir string, logDir string, name string) error {
	cmd := exec.Command(e.cowerPath, "--download", name)
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

// Make runs makepkg on a fetched command.
func (e *Exec) Make(dir string, logDir string, name string, skippgpcheck bool) (pkgPath string, err error) {
	cmd := exec.Command(e.makePkgPath, "--syncdeps", name)
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
