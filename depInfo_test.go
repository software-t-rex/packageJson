/*
Copyright © 2023 Jonathan Gotti <jgotti at jgotti dot org>
SPDX-FileType: SOURCE
SPDX-License-Identifier: MIT
SPDX-FileCopyrightText: 2023 Jonathan Gotti <jgotti@jgotti.org>
*/

package packageJson

import (
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestPackageJson_SatisfyWorskpaceDep(t *testing.T) {
	overridePkgDir := func(p *PackageJSON, dirJoin string) {
		p.Dir = filepath.Join(p.Dir, dirJoin)
		p.file = filepath.Join(p.Dir, filepath.Base(p.file))
	}
	type args struct {
		path       string // path to package
		pkgDirJoin string // override package directory
		depDirJoin string // override dependency directory
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{"work with workspace protocol", args{"wsFoo.json", "", ""}, true, false},
		{"don't work if path mismatch", args{"devBaz.json", "", ""}, false, true},                                               // "devBaz": "local:../devBaz" -> not rewriting dir should not work
		{"don't work if pckg is outside of the workspace", args{"devBaz.json", "../devBaz", ""}, false, true},                   // "devBaz": "local:../devBaz" -> package outside the workspace should not work
		{"work with file protocol inside the workspace", args{"devBaz.json", "packages/devBaz", "packages/fooer"}, true, false}, // "devBaz": "local:../devBaz" -> both resolving in the workspace should work
		{"Don't work if version mismatch", args{"devFoo.json", "", ""}, false, true},
		// @TODO add missing test cases
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPkg, err := Read("./testdata/pkgStrings.json")
			assert.NilError(t, err)
			// override testPkgDir to force dependency dir
			if tt.args.depDirJoin != "" {
				overridePkgDir(testPkg, tt.args.depDirJoin)
			}
			pkg, err := Read(filepath.Join("./testdata", tt.args.path))
			assert.NilError(t, err, "error reading package %s", tt.args.path)
			// override pkgDir
			if tt.args.pkgDirJoin != "" {
				overridePkgDir(pkg, tt.args.pkgDirJoin)
			}

			dep, ok := testPkg.GetDepencyInfoFor(pkg.Name)
			assert.Assert(t, ok, "can't find depency info for %s", pkg.Name)
			satisfy, err := pkg.SatisfyWorskpaceDep(dep, "./testdata")
			assert.Assert(t, satisfy == tt.want, "satisfyWorskpaceDep(%+v) = %v, want %v, err %v", dep, tt.want, satisfy, err)
			if tt.wantErr && err == nil {
				t.Fatalf("Should return an error")
			} else if !tt.wantErr && err != nil {
				t.Fatalf("Should not return an error and get %s", err)
			}
		})
	}
}
