package main

import (
	"os"
	"path"

	"github.com/pkg/errors"
)

// AurPackage is a data holder for what we know about a package
type AurPackage struct {
	// name of the package
	Name string
	// root of the package directory
	Root string

	// set after a successful make
	PkgPath string
}

// NewAurpackage creates a new AurPackage.  The next step is to call PreparePackageDir.
func NewAurPackage(root, name string) *AurPackage {
	return &AurPackage{
		Name: name,
		Root: path.Join(root, name),
	}
}

// PreparePackageDir creates a package directory we can download to later.
func (a *AurPackage) PreparePackageDir() error {
	if err := os.RemoveAll(a.Root); err != nil {
		return errors.Wrapf(err, "Unable to clean %q", a.Root)
	}
	if err := os.MkdirAll(a.Logs(), dirMode); err != nil {
		return err
	}
	if err := os.MkdirAll(a.Build(), dirMode); err != nil {
		return err
	}
	return nil
}

// Logs returns the the directory where we should put log files.
func (a *AurPackage) Logs() string {
	return path.Join(a.Root, "logs")
}

// Build return the directory where we should put build files.
func (a *AurPackage) Build() string {
	return path.Join(a.Root, "build")
}
