package packageJson

import (
	"path/filepath"
	"sort"
	"testing"

	"gotest.tools/v3/assert"
)

func TestRead(t *testing.T) {

	pkgUnknwon, err := Read("./testdata/missing.json")
	assert.ErrorContains(t, err, "no such file or directory")
	assert.Assert(t, pkgUnknwon == nil)
	pkgInvalid, err := Read("./testdata/invalid.json")
	assert.ErrorIs(t, err, ErrInvalidJson)
	assert.Assert(t, pkgInvalid == nil)
	pkgStrings, err := Read("./testdata/pkgStrings.json")
	assert.NilError(t, err)
	assert.Assert(t, pkgStrings != nil)
	pkgObjects, err := Read("./testdata/pkgObjects.json")
	assert.NilError(t, err)
	assert.Assert(t, pkgObjects != nil)
	workspaces, err := Read("./testdata/workspaces.json")
	assert.NilError(t, err)
	assert.Assert(t, workspaces != nil)
	mixedFundings, err := Read("./testdata/fundingAsMixedArray.json")
	assert.NilError(t, err)
	assert.Assert(t, mixedFundings != nil)

	type args struct {
		pkg *PackageJSON
	}
	tests := []struct {
		name   string
		args   args
		assert func(*testing.T, *PackageJSON)
	}{
		{
			"read corectly as string",
			args{pkg: pkgStrings},
			func(t *testing.T, pkg *PackageJSON) {
				assert.Assert(t, pkg.Name == "fooer")
				assert.Assert(t, pkg.Version == "1.2.3")
				assert.Assert(t, pkg.Description == "A packaged fooer for fooing foos")
				assert.Assert(t, pkg.Main == "fooer.js")
				assert.Assert(t, pkg.Man == "./man/doc.1")
				assert.Assert(t, pkg.Bin == "path/to/bin")
				assert.Assert(t, pkg.Author == "Barney Rubble <b@rubble.com> (http://barnyrubble.tumblr.com/)")
				assert.Assert(t, pkg.Funding == "http://example.com/donate")
			},
		},
		{
			"read correctly as Objet",
			args{pkg: pkgObjects},
			func(t *testing.T, p *PackageJSON) {
				assert.DeepEqual(t, p.Author, map[string]any{
					"name":  string("Barney Rubble"),
					"email": string("b@rubble.com"),
					"url":   string("http://barnyrubble.tumblr.com/"),
				})
				assert.DeepEqual(t, p.Bin, map[string]any{
					"my-program":       string("./path/to/program"),
					"my-other-program": string("./path/to/other/program"),
				})
				assert.DeepEqual(t, p.Funding, map[string]any{
					"type": string("individual"),
					"url":  string("http://example.com/donate"),
				})
				assert.DeepEqual(t, p.Man, []any{
					string("./man/foo.1"),
					string("./man/bar.1"),
				})
			},
		},
		{
			"read correctly workspaces",
			args{pkg: workspaces},
			func(t *testing.T, p *PackageJSON) {
				assert.DeepEqual(t, p.Workspaces, []string{
					"apps/*",
					"packages/**",
					"!**/tests/**",
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { tt.assert(t, tt.args.pkg) })
	}
}

func TestPackageJSON_GetMergedDependencies(t *testing.T) {
	pkg, err := Read("./testdata/pkgStrings.json")
	assert.NilError(t, err)
	assert.DeepEqual(t, pkg.GetMergedDependencies(), map[string]string{
		"optFoo": "~7.8.9",
		"devFoo": "<=4.5.6",
		"devBar": "npm:0.0.1",
		"devBaz": "file:../devBaz",
		"foo":    "^1.2.3",
		"wsFoo":  "workspace:*",
	})
}

func TestPackageJSON_GetAvailableTasks(t *testing.T) {
	pkg, err := Read("./testdata/pkgStrings.json")
	assert.NilError(t, err)
	availables := pkg.GetAvailableTasks()
	sort.Strings(availables)
	assert.DeepEqual(t, availables, []string{"start", "test"})
	pkg, err = Read("./testdata/fundingAsMixedArray.json")
	assert.NilError(t, err)
	assert.Assert(t, len(pkg.GetAvailableTasks()) == 0)
}

func TestPackageJSON_HasTask(t *testing.T) {
	pkg, err := Read("./testdata/pkgStrings.json")
	assert.NilError(t, err)
	assert.Assert(t, pkg.HasTask("start"))
	pkg, err = Read("./testdata/fundingAsMixedArray.json")
	assert.NilError(t, err)
	assert.Assert(t, !pkg.HasTask("start"))

}

func TestPackageJSON_FilterWorkspaceDirs(t *testing.T) {
	rootDir, _ := filepath.Abs("./testdata/")
	pkg, err := Read("./testdata/workspaces.json")
	testDirs := []string{
		"apps/test",
		"apps/test/resources",
		"packages/foo",
		"packages/bar",
		"packages/bar/baz",
		"apps/tests/doc",
		"packages/bar/tests",
	}
	assert.NilError(t, err)
	assert.DeepEqual(t, pkg.FilterWorkspaceDirs(testDirs, false), []string{
		"apps/test",
		"packages/foo",
		"packages/bar",
		"packages/bar/baz",
	})
	assert.DeepEqual(t, pkg.FilterWorkspaceDirs(testDirs, true), []string{
		filepath.Join(rootDir, "apps/test"),
		filepath.Join(rootDir, "packages/foo"),
		filepath.Join(rootDir, "packages/bar"),
		filepath.Join(rootDir, "packages/bar/baz"),
	})
}

func TestPackageJSON_GetDepencyInfoFor(t *testing.T) {
	pkg, _ := Read("./testdata/pkgStrings.json")
	type want struct {
		Key          string
		Name         string
		VersionRange string
		Protocol     string
	}
	compareDepInfo := func(t *testing.T, i PackageJsonDepInfo, w want) {
		if w.Key == "" {
			assert.DeepEqual(t, i, PackageJsonDepInfo{})
			return
		}
		assert.Assert(t, i.FromName == "fooer", "expected fooer, got %s", i.FromName)
		assert.Assert(t, i.FromVersion == "1.2.3", "expected 1.2.3, got %s", i.FromVersion)
		assert.Assert(t, i.FromFile == pkg.file, "expected %s, got %s", pkg.file, i.FromFile)
		assert.Assert(t, i.Name == w.Name, "expected %s, got %s", w.Name, i.Name)
		assert.Assert(t, i.VersionRange == w.VersionRange, "expected %s, got %s", w.VersionRange, i.VersionRange)
		assert.Assert(t, i.Key == w.Key, "expected %s, got %s", w.Key, i.Key)
		assert.Assert(t, i.Protocol == w.Protocol, "expected %s, got %s", w.Protocol, i.Protocol)
	}
	tests := []struct {
		name   string
		want   want
		wantOk bool
	}{
		{"retrieving dependency info", want{"dependencies", "foo", "^1.2.3", ""}, true},
		{"retrieving devDependency info", want{"devDependencies", "devFoo", "<=4.5.6", ""}, true},
		{"retrieving devDependency info", want{"devDependencies", "devBar", "0.0.1", "npm"}, true},
		{"retrieving devDependency info", want{"devDependencies", "devBaz", "../devBaz", "file"}, true},
		{"retrieving optionalDependency info", want{"optionalDependencies", "optFoo", "~7.8.9", ""}, true},
		{"retrieving peerDependency info", want{"peerDependencies", "peerFoo", "10.11.x", ""}, true},
		{"retrieving dependency with protocol", want{"dependencies", "wsFoo", "*", "workspace"}, true},
		{"retrieving unknown dependency", want{"", "Unknwon", "", ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := pkg.GetDepencyInfoFor(tt.want.Name)
			compareDepInfo(t, got, tt.want)
			assert.Equal(t, got1, tt.wantOk)
		})
	}
}
