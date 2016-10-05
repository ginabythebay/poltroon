package poltroon

import (
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
)

// AurPackage is a data holder for what we know about a package
type AurPackage struct {
	// name of the package
	Name string
	// CurrentVersion is the version of this package currently installed.
	CurrentVersion string
	// NextVersion is the available version of this package
	NextVersion string

	// url to fetch the current snapshot
	SnapshotURL string

	// root of the package directory
	Root string

	// set after a successful make
	PkgPath string
}

// NewAurPackage creates a new AurPackage.  The next step is to call PreparePackageDir.
func NewAurPackage(root, name, currentVersion, nextVersion, snapshotURL string) *AurPackage {
	return &AurPackage{
		Name:           name,
		CurrentVersion: currentVersion,
		NextVersion:    nextVersion,
		SnapshotURL:    snapshotURL,
		Root:           path.Join(root, name),
	}
}

func (a *AurPackage) String() string {
	return fmt.Sprintf(":: %s %s -> %s", a.Name, a.CurrentVersion, a.NextVersion)
}

// PreparePackageDir creates a package directory we can download to later.
func (a *AurPackage) PreparePackageDir(perm os.FileMode) error {
	if err := os.RemoveAll(a.Root); err != nil {
		return errors.Wrapf(err, "Unable to clean %q", a.Root)
	}
	if err := os.MkdirAll(a.Logs(), perm); err != nil {
		return err
	}
	if err := os.MkdirAll(a.Build(), perm); err != nil {
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
