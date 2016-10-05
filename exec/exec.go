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
	pacmanPath  string
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
	pacmanPath, err := findPgm("pacman")
	if err != nil {
		return nil, err
	}
	makePkgPath, err := findPgm("makepkg")
	if err != nil {
		return nil, err
	}

	return &Exec{pacmanPath, makePkgPath}, nil
}

// QueryForeignPackages returns all packages that are installed but
// don't exist in the sync databases (e.g. they were installed via
// pacman --upgrade)
func (e *Exec) QueryForeignPackages() ([]VersionedPackage, error) {
	cmd := exec.Command(e.pacmanPath, "--query", "--foreign")
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "Getting stdout for pacman")
	}
	if err = cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "starting pacman")
	}
	pkgs := []VersionedPackage{}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		// Example line:
		// ledger 3.1.1-3
		var name, version string
		_, err = fmt.Sscan(line, &name, &version)
		if err != nil {
			return nil, errors.Wrapf(err, "Unable to parse [%s]", line)
		}
		pkgs = append(pkgs, VersionedPackage{name, version})
	}
	if err = cmd.Wait(); err != nil {
		return nil, errors.Wrap(err, "executing pacman")
	}
	return pkgs, nil
}

// VersionedPackage is just a package name with a version.
type VersionedPackage struct {
	Name    string
	Version string
}

// Make runs makepkg on a fetched command.  If it is successful, a.PkgPath will be set
// to the package we built.
func (e *Exec) Make(a *poltroon.AurPackage, skippgpcheck bool) error {
	cmd := exec.Command(e.makePkgPath, "--syncdeps", a.Name)
	if skippgpcheck {
		cmd.Args = append(cmd.Args, "--skippgpcheck")
	}
	cmd.Dir = path.Join(a.Build(), a.Name)
	stdout, err := os.Create(path.Join(a.Logs(), "make.out"))
	if err != nil {
		return errors.Wrapf(err, "Making %s", a.Name)
	}
	defer stdout.Close()
	cmd.Stdout = stdout

	stderr, err := os.Create(path.Join(a.Logs(), "make.err"))
	if err != nil {
		return errors.Wrapf(err, "Making %s", a.Name)
	}
	defer stderr.Close()
	cmd.Stderr = stderr

	if err = cmd.Run(); err != nil {
		return errors.Wrapf(err, "running makepkg for %s.  See %s", a.Name, a.Logs())
	}

	matches, err := filepath.Glob(path.Join(cmd.Dir, "*.pkg.*"))
	if err != nil {
		return errors.Wrapf(err, "globbing makepkg for %s.  See %s", a.Name, cmd.Dir)
	}
	if len(matches) != 1 {
		return errors.Errorf("Expected exactly 1 match but got this instead: %v", matches)
	}

	a.PkgPath = matches[0]
	return nil
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
