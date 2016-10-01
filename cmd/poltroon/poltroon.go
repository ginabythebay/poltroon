package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ginabythebay/poltroon"
	"github.com/ginabythebay/poltroon/exec"
	"github.com/urfave/cli"
)

const dirMode = os.ModeDir | 0770

var (
	// work queue of things to fetch
	fetchChan = make(chan *poltroon.AurPackage)
	// work queue of things to make
	makeChan = make(chan *poltroon.AurPackage)
)

func main() {
	app := cli.NewApp()
	app.Usage = strings.TrimSpace(`
Foolishly upgrade AUR packages.

1. Runs cower -u to find already-installed packages that are out of
   date, prints out the results.
2. Asks if the user wants to proceed.  Exits if they don't.
3. Starts a two-stage pipeline.
4. In the first stage, we run cower -d to download the package (default it two workers).
5. In the second state, we run makepkg -s to build the package files.
5. At the end, we print out the command the user can run to install the packages.

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
			Value: 2,
			Usage: "Number of concurrent makers",
		},
		cli.BoolFlag{
			Name:  "skippgpcheck",
			Usage: "Turn off pgp checks",
		},
		cli.BoolFlag{
			Name:  "noconfirm",
			Usage: "Don't ask if the user wants to proceed.",
		},
	}
	app.Action = func(c *cli.Context) error {
		start := time.Now()

		root, err := getRoot()
		if err != nil {
			fatal(err)
		}

		exec, err := exec.Find()
		if err != nil {
			fatal(err)
		}
		aurPkgs, err := exec.QueryUpdates(root)
		if err != nil {
			fatal(err)
		}

		var waitGroup sync.WaitGroup

		if len(aurPkgs) == 0 {
			fmt.Println("Nothing to update!")
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

		for i := 0; i < c.Int("makers"); i++ {
			go func() {
				for pkg := range makeChan {
					output(fmt.Sprintf("%s: beginning make...", pkg.Name))
					pkgPath, err := exec.Make(pkg.Build(), pkg.Logs(), pkg.Name, c.Bool("skippgpcheck"))
					if err != nil {
						output(fmt.Sprintf("%s: failed to make due to %+v", pkg.Name, err))
						waitGroup.Done()
						continue
					}
					pkg.PkgPath = pkgPath
					output(fmt.Sprintf("%s: successfully made", pkg.Name))
					waitGroup.Done()
				}
			}()
		}

		for i := 0; i < c.Int("fetchers"); i++ {
			go func() {
				for pkg := range fetchChan {
					output(fmt.Sprintf("%s: beginning fetch...", pkg.Name))
					err := pkg.PreparePackageDir(dirMode)
					if err != nil {
						output(fmt.Sprintf("%s: failed to fetch due to %+v", pkg.Name, err))
						waitGroup.Done()
						continue
					}
					err = exec.Fetch(pkg.Build(), pkg.Logs(), pkg.Name)
					if err != nil {
						output(fmt.Sprintf("%s: failed to fetch due to %+v", pkg.Name, err))
						waitGroup.Done()
						continue
					}
					output(fmt.Sprintf("%s: successfully fetched", pkg.Name))
					makeChan <- pkg
				}
			}()
		}

		waitGroup.Add(len(aurPkgs))
		for _, a := range aurPkgs {
			fetchChan <- a
		}

		waitGroup.Wait()

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

		fmt.Printf("\nYou can now run:\n")
		if len(good) != 0 {
			fmt.Printf("    sudo pacman -U --noconfirm %s\n", strings.Join(good, " "))
		}
		fmt.Printf("    rm -rf %s/*\n", root)

		return nil
	}

	app.Run(os.Args)
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