/*
Copyright © 2023 Jonathan Gotti <jgotti at jgotti dot org>
SPDX-FileType: SOURCE
SPDX-License-Identifier: MIT
SPDX-FileCopyrightText: 2023 Jonathan Gotti <jgotti@jgotti.org>
*/

package packageJson

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/bmatcuk/doublestar/v4"
)

type PackageJsonDepInfo struct {
	Name string
	// can be a version range or tarball, or git url as defined in package.json
	VersionRange string
	// one of dependecies, devDependecies, peerDependecies, optionalDependecies
	Key string
	// contains protocol if defined
	Protocol string
	// keep info on the package that require this
	FromName    string
	FromVersion string
	FromFile    string
}

func newDependencyInfo(p *PackageJSON, moduleName string) (i PackageJsonDepInfo, ok bool) {
	var depKey, depVal string
	if depVal, ok = p.Dependencies[moduleName]; ok {
		depKey = "dependencies"
	} else if depVal, ok = p.DevDependencies[moduleName]; ok {
		depKey = "devDependencies"
	} else if depVal, ok = p.OptionalDependencies[moduleName]; ok {
		depKey = "optionalDependencies"
	} else if depVal, ok = p.PeerDependencies[moduleName]; ok {
		depKey = "peerDependencies"
	}

	if !ok {
		return
	}
	i.Name = moduleName
	i.Key = depKey
	i.FromName = p.Name
	i.FromVersion = p.Version
	i.FromFile = p.file

	parts := strings.Split(depVal, ":")
	if len(parts) == 1 {
		i.VersionRange = parts[0]
	} else {
		i.Protocol = parts[0]
		i.VersionRange = strings.Join(parts[1:], ":")
	}
	return
}

var ErrSatisfaction error = errors.New("SatisfyWorskpaceDep failed: ")
var ErrMismatchName error = fmt.Errorf("%wpackage name mismatch", ErrSatisfaction)
var ErrMismatchPath error = fmt.Errorf("%wpackage path mismatch dependency path", ErrSatisfaction)
var ErrParseVersion error = fmt.Errorf("%wpackage version check failed", ErrSatisfaction)
var ErrMissingWorkspaceInfo error = fmt.Errorf("%wcannot satisfy workspace protocol dependency without a workspace directory", ErrSatisfaction)
var ErrOutsideWokspace error = fmt.Errorf("%wcan't satisfy a dependency being outside workspace", ErrSatisfaction)
var ErrRemoteProtocol = fmt.Errorf("%wcannot satisfy remote protocol", ErrSatisfaction)

// workspacesRootDir is optional and is used to determine that the package is withing the root of the workspace directory
// if not explicitly specified it will default to the directory of dep.FromFile
// if false you should also get an error explaining the reason for the failure
// you can also receive error while assuming to return true in certain cases
func (p *PackageJSON) SatisfyWorskpaceDep(dep PackageJsonDepInfo, workspacesRootDir string) (bool, error) {
	if p.Name != dep.Name {
		return false, fmt.Errorf("%w: %s!= %s", ErrMismatchName, p.Name, dep.Name)
	}
	// if empty set workspaceRootDir as dependecy package directory
	if workspacesRootDir == "" {
		if dep.Protocol == "workspace" {
			return false, ErrMissingWorkspaceInfo
		}
		workspacesRootDir = filepath.Dir(dep.FromFile)
	} else if !filepath.IsAbs(workspacesRootDir) {
		workspacesRootDirAbs, err := filepath.Abs(workspacesRootDir)
		if err != nil {
			return false, err
		}
		workspacesRootDir = workspacesRootDirAbs
	}

	// check package dir is in same workspace than the dep
	if ok, err := doublestar.Match(filepath.Join(workspacesRootDir, "**"), p.Dir); !ok {
		if err != nil {
			return false, fmt.Errorf("%w: can't check dependency is inside workspace", err)
		}
		return false, ErrOutsideWokspace
	}

	// workspace protocol => considered ok no more check
	if dep.Protocol == "workspace" {
		return true, nil
	}

	// local protocols we should look at the resolved path
	if dep.Protocol == "file" || dep.Protocol == "link" || dep.Protocol == "portal" {
		depPath := filepath.Join(filepath.Dir(dep.FromFile), dep.VersionRange) // in such case versionRange is a path
		if depPath != p.Dir {
			return false, fmt.Errorf("%w: Package "+dep.Name+" path does not match workspace package "+p.Name, ErrMismatchPath)
		}
		return true, nil
	}

	// any other protocols than defaults should be considered remote and so are not ok
	if dep.Protocol != "" && dep.Protocol != "npm" {
		return false, ErrRemoteProtocol
	}

	// at this point We need to check the version range
	if dep.VersionRange == "*" || dep.VersionRange == "^" || dep.VersionRange == "~" {
		return true, nil
	}
	constraint, errc := semver.NewConstraint(dep.VersionRange)
	if errc != nil {
		// if we can't parse the version, we can't check the range and will assume it is ok but will return an error
		return true, fmt.Errorf("%wcan't parse version: %s", ErrParseVersion, dep.VersionRange)
	}
	version, errv := semver.NewVersion(p.Version)
	if errv != nil {
		// if we can't parse the version, we can't check the range and will assume it is ok but will return an error
		return true, fmt.Errorf("%wcan't parse version: %s", ErrParseVersion, p.Version)
	}
	versionOk := constraint.Check(version)
	if versionOk {
		return true, nil
	}
	return false, ErrParseVersion
}
