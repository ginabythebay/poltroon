package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ginabythebay/poltroon"
	"github.com/ginabythebay/poltroon/alpm"
	"github.com/ginabythebay/poltroon/aur"
	"github.com/ginabythebay/poltroon/exec"
	"github.com/ginabythebay/poltroon/tar"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const dirMode = os.ModeDir | 0770

var (
	// work queue of things to fetch
	fetchChan = make(chan *poltroon.AurPackage)
	// work queue of things to make
	makeChan = make(chan *poltroon.AurPackage)

	updateState *poltroon.UpdateState
)

func main() {
	app := cli.NewApp()
	app.Usage = strings.TrimSpace(`
Foolishly upgrade AUR packages.

1. Finds foreign packages by running packman --query --foreign and
   then consults the AUR for newer versions.  Prints out all packages
   found to have newer versions.
2. Asks if the user wants to proceed.  Exits if they don't.
3. Starts a two-stage pipeline.
4. In the first stage, we run download the package (defaulting it 2 workers).
5. In the second state, we run makepkg -s to build the package files.
6. At the end, we print out the command the user can run to install the packages.

All the action happens in /tmp/poltroon/ with a sub-directory for each package and a logs directory within that that can be inspected.
`)
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "fetchers",
			Value: 2,
			Usage: "Number of concurrent fetchers",
		},
		cli.IntFlag{
			Name:  "makers",
			Value: 3,
			Usage: "Number of concurrent makers",
		},
		cli.BoolFlag{
			Name:  "licenses",
			Usage: "Print out license information and exit.",
		},
		cli.BoolFlag{
			Name:  "skippgpcheck",
			Usage: "Turn off pgp checks",
		},
		cli.BoolFlag{
			Name:  "noconfirm",
			Usage: "Don't ask if the user wants to proceed.",
		},
		cli.BoolFlag{
			Name:  "quiet",
			Usage: "Don't print progress updates.",
		},
	}
	app.Action = func(c *cli.Context) error {
		if c.Bool("licenses") {
			printAllLicenses()
			os.Exit(0)
		}

		start := time.Now()

		root, err := getRoot()
		if err != nil {
			fatal(err)
		}

		exec, err := exec.Find()
		if err != nil {
			fatal(err)
		}
		aurPkgs, err := queryUpdates(exec, root)
		if err != nil {
			fatal(err)
		}

		if len(aurPkgs) == 0 {
			elapsed := time.Since(start)
			fmt.Printf("Nothing to update!  Exiting in %s\n", elapsed)
			os.Exit(0)
		}

		for _, a := range aurPkgs {
			fmt.Println(a)
		}

		fmt.Println()
		if c.Bool("noconfirm") {
			fmt.Println("Proceeding to update all packages because --noconfirm was set...")
		} else {
			msg := fmt.Sprintf("Do you want to update these %d packages?", len(aurPkgs))
			if !askForConfirmation(msg) {
				os.Exit(0)
			}
		}
		fmt.Println()

		updateState = poltroon.NewUpdateState(len(aurPkgs))

		if !c.Bool("quiet") {
			go func() {
				for s := range updateState.Makes {
					fmt.Print(s)
				}
			}()
		}

		// Start our asynchronous pipeline
		startFetchers(exec, c.Int("fetchers"))
		startMakers(exec, c.Int("makers"), c.Bool("skippgpcheck"))

		// Push things into the pipeline here
		for _, a := range aurPkgs {
			fetchChan <- a
		}

		updateState.Wait()

		var good, bad []string
		for _, pkg := range aurPkgs {
			if pkg.PkgPath == "" {
				bad = append(bad, pkg.Name)
			} else {
				good = append(good, pkg.PkgPath)
			}
		}

		elapsed := time.Since(start)

		fmt.Println()
		for _, b := range bad {
			fmt.Printf("***Error processing %s***\n", b)
		}
		if len(bad) != 0 {
			fmt.Println()
		}

		if len(bad) == 0 && len(good) == 0 {
			fmt.Printf("Found nothing to do in %s\n", elapsed)
		} else {
			fmt.Printf("Created %d packages in %s\n", len(good), elapsed)
		}

		if len(good) != 0 {
			fmt.Printf("\nAfter you look in %s and verify it looks good, run:\n", root)
			fmt.Printf("    sudo pacman -U --noconfirm %s\n", strings.Join(good, " "))
		}
		fmt.Printf("\nTo clean up, run\n")
		fmt.Printf("    rm -rf %s/*\n", root)

		return nil
	}

	app.Run(os.Args)
}

func printAllLicenses() {
	PrintLicense("poltroon", "LICENSE")
	for _, n := range poltroon.AssetNames() {
		if n == "LICENSE" {
			continue
		}
		PrintLicense(n, n)
	}
}

func PrintLicense(titleName, assetName string) {
	license, err := poltroon.Asset(assetName)
	if err != nil {
		fmt.Printf("Error finding asset %q: %+v", assetName, err)
		os.Exit(-1)
	}
	fmt.Println()
	fmt.Printf("LICENSE For %s:\n%s\n", titleName, license)
}

func startFetchers(e *exec.Exec, fetcherCnt int) {
	for i := 0; i < fetcherCnt; i++ {
		go func() {
			// Fetch each package.  When we finish with a package we
			// either pass it onto the next stage of the pipeline or
			// we error mark it finished.
			for pkg := range fetchChan {
				fetchPackage(e, pkg)
			}
		}()
	}
}

func fetchPackage(e *exec.Exec, pkg *poltroon.AurPackage) {
	var err error
	defer func() {
		if err != nil {
			output(fmt.Sprintf("%s: failed to fetch due to %+v", pkg.Name, err))
			updateState.Finished(pkg.Name)
			return
		}
		makeChan <- pkg
	}()

	err = pkg.PreparePackageDir(dirMode)
	if err != nil {
		return
	}

	resp, err := http.Get(pkg.SnapshotURL)
	if err != nil {
		err = errors.Wrapf(err, "%s: fetching %s", pkg.Name, pkg.SnapshotURL)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = errors.Errorf("%s: fetching %s unexpected status %d/%s", pkg.Name, pkg.SnapshotURL, resp.StatusCode, resp.Status)
		return
	}

	ungzipper, err := gzip.NewReader(resp.Body)
	if err != nil {
		err = errors.Wrapf(err, "%s: decompressing", pkg.Name, pkg.SnapshotURL)
		return
	}
	err = tar.ExtractAll(ungzipper, pkg.Build())
	if err != nil {
		err = errors.Wrapf(err, "%s: extracting", pkg.Name, pkg.SnapshotURL)
		return
	}
}

func startMakers(e *exec.Exec, makerCnt int, skipPgpCheck bool) {
	for i := 0; i < makerCnt; i++ {
		go func() {
			for pkg := range makeChan {
				makePackage(e, skipPgpCheck, pkg)
			}
		}()
	}
}

func makePackage(e *exec.Exec, skipPgpCheck bool, pkg *poltroon.AurPackage) {
	updateState.StartMake(pkg.Name)
	defer updateState.Finished(pkg.Name)

	err := e.Make(pkg, skipPgpCheck)
	if err != nil {
		output(fmt.Sprintf("%s: failed to make due to %+v", pkg.Name, err))
		return
	}
}

func queryUpdates(e *exec.Exec, root string) ([]*poltroon.AurPackage, error) {
	foreign, err := e.QueryForeignPackages()
	if err != nil {
		return nil, errors.Wrap(err, "queryUpdates")
	}

	names := []string{}
	for _, f := range foreign {
		names = append(names, f.Name)
	}

	allInfos, err := aur.GetInfos(names)
	if err != nil {
		fatal(fmt.Sprintf("%+v: Get aur info for names", err))
	}

	result := []*poltroon.AurPackage{}
	for _, f := range foreign {
		info, ok := allInfos[f.Name]
		if ok && alpm.VerCmp(f.Version, info.Version) < 0 {
			pkg := poltroon.NewAurPackage(root, f.Name, f.Version, info.Version, info.SnapshotURL)
			result = append(result, pkg)
		}
	}
	return result, nil
}

var outputMutex sync.Mutex

func output(s string) {
	outputMutex.Lock()
	fmt.Println(s)
	outputMutex.Unlock()
}

func fatal(v ...interface{}) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}

func getRoot() (string, error) {
	root := path.Join(os.TempDir(), "poltroon")
	err := os.MkdirAll(root, dirMode)
	return root, err
}

func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/N]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" || response == "" {
			return false
		}
	}
}
