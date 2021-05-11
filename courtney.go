package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dave/courtney/scanner"
	"github.com/dave/courtney/shared"
	"github.com/dave/courtney/tester"
	"github.com/dave/patsy"
	"github.com/dave/patsy/vos"
	"github.com/pkg/errors"
)

func main() {
	// notest
	env := vos.Os()

	enforceFlag := flag.Bool("e", false, "Enforce 100% code coverage")
	verboseFlag := flag.Bool("v", false, "Verbose output")
	reportFlag := flag.Bool("r", false, "Print each package being tested")
	shortFlag := flag.Bool("short", false, "Pass the short flag to the go test command")
	timeoutFlag := flag.String("timeout", "", "Pass the timeout flag to the go test command")
	outputFlag := flag.String("o", "", "Override coverage file location")
	testArgsFlag := new(argsValue)
	flag.Var(testArgsFlag, "t", "Argument to pass to the 'go test' command. Can be used more than once.")
	loadFlag := flag.String("l", "", "Load coverage file(s) instead of running 'go test'")
	excludeErrNoReturnParamFlag := flag.Bool("excludenoreturn", false, "Exclude error blocks in functions with no return params")
	coverPkgFlag := new(argsValue)
	flag.Var(coverPkgFlag, "coverpkg", "Specify which packages to cover")
	excludePkgsFlag := new(argsValue)
	flag.Var(excludePkgsFlag, "ex", "Argument to exclude packages from the cover profile.")

	flag.Parse()
	args := flag.Args()

	// any args after a "--" arg will be considered args for `go test`
	testArgs := testArgsFlag.args
	pkgArgs := make([]string, 0, len(args))
	foundTestArgs := false
	for _, arg := range args {
		if strings.TrimSpace(arg) == "--" {
			foundTestArgs = true
			continue
		}
		if foundTestArgs {
			testArgs = append(testArgs, arg)
		} else {
			pkgArgs = append(pkgArgs, arg)
		}
	}

	// coverpkg flag supports both multiple flags and csv
	coverPkgs := make([]string, 0)
	for _, coverPkg := range coverPkgFlag.args {
		coverPkgs = append(coverPkgs, strings.Split(coverPkg, ",")...)
	}

	// excludepkg flag supports both multiple flags and csv
	excludePkgs := make([]string, 0)
	for _, excludePkg := range excludePkgsFlag.args {
		excludePkgs = append(excludePkgs, strings.Split(excludePkg, ",")...)
	}

	setup := &shared.Setup{
		Env:           env,
		Paths:         patsy.NewCache(env),
		Enforce:       *enforceFlag,
		Verbose:       *verboseFlag,
		ReportTestRun: *reportFlag,
		Short:         *shortFlag,
		Timeout:       *timeoutFlag,
		Output:        *outputFlag,
		Options: shared.Options{
			ExcludeErrNoReturnParam: *excludeErrNoReturnParamFlag,
		},
		TestArgs:    testArgs,
		CoverPkgs:   coverPkgs,
		ExcludePkgs: excludePkgs,
		Load:        *loadFlag,
	}
	if err := Run(setup, pkgArgs); err != nil {
		fmt.Printf("%+v", err)
		os.Exit(1)
	}
}

// Run initiates the command with the provided setup
func Run(setup *shared.Setup, pkgArgs []string) error {
	var err error

	if err := setup.Parse(pkgArgs); err != nil {
		return errors.Wrapf(err, "Parse")
	}

	// normalize coverpkgs paths
	if len(setup.CoverPkgs) > 0 {
		setup.CoverPkgs, err = setup.ParsePkgArgs(setup.CoverPkgs)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// normalize excludepkgs paths
	if len(setup.ExcludePkgs) > 0 {
		setup.ExcludePkgs, err = setup.ParsePkgArgs(setup.ExcludePkgs)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	s := scanner.New(setup)
	if err := s.LoadProgram(); err != nil {
		return errors.Wrapf(err, "LoadProgram")
	}
	if err := s.ScanPackages(); err != nil {
		return errors.Wrapf(err, "ScanPackages")
	}

	t := tester.New(setup)
	if setup.Load == "" {
		if err := t.Test(); err != nil {
			return errors.Wrapf(err, "Test")
		}
	} else {
		if err := t.Load(); err != nil {
			return errors.Wrapf(err, "Load")
		}
	}
	if err := t.ProcessExcludes(s.Excludes); err != nil {
		return errors.Wrapf(err, "ProcessExcludes")
	}
	if err := t.Save(); err != nil {
		return errors.Wrapf(err, "Save")
	}
	if err := t.Enforce(); err != nil {
		return errors.Wrapf(err, "Enforce")
	}

	return nil
}

type argsValue struct {
	args []string
}

var _ flag.Value = (*argsValue)(nil)

func (v *argsValue) String() string {
	// notest
	if v == nil {
		return ""
	}
	return strings.Join(v.args, " ")
}
func (v *argsValue) Set(s string) error {
	// notest
	v.args = append(v.args, s)
	return nil
}
