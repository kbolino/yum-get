package main

import (
	"compress/bzip2"
	"compress/gzip"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/kbolino/yum-get/yum"
)

var (
	flagList    = flag.Bool("list", false, "list packages in repository instead of downloading")
	flagRepo    = flag.String("repo", "", "URL of Yum repositoryto use")
	flagVerbose = flag.Bool("verbose", false, "enable debugging info to stderr")
)

var (
	repomdRelURL = mustParseURL("repodata/repomd.xml")
	rePkgName    = regexp.MustCompile(`^(.+)-([^-]+)-([^-]+)$`)
)

func main() {
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "USAGE: %s [options] -repo URL -list\n", os.Args[0])
		fmt.Fprintf(out, "       %s [options] -repo URL PKG ...\n", os.Args[0])
		fmt.Fprintln(out, "Lists or downloads RPM packages from a Yum repository.")
		fmt.Fprintln(out, "Specify each PKG to download as name-ver-rel")
		fmt.Fprintln(out, "\nOPTIONS:")
		flag.PrintDefaults()
	}
	flag.Parse()
	os.Exit(run())
}

func run() int {
	pkgsToGet := flag.Args()
	if (!*flagList && len(pkgsToGet) == 0) || (*flagList && len(pkgsToGet) != 0) {
		errorf("must specify exactly one of -list or package names to download")
		return 1
	}
	repoURL, err := url.Parse(*flagRepo)
	if err != nil {
		errorf("invalid repo URL: %s", err)
		return 1
	}
	repomdURL := repoURL.ResolveReference(repomdRelURL)
	debugf("downloading repo metadata from %s", repomdURL)
	response, err := get(repomdURL)
	if err != nil {
		errorf("failed to download repo metadata: %s", err)
		return 1
	}
	decoder := xml.NewDecoder(response.Body)
	var repoMD yum.RepoMD
	if err := decoder.Decode(&repoMD); err != nil {
		errorf("failed to decode repo metadata as XML: %s", err)
		return 1
	}
	primaryHref := ""
	for _, data := range repoMD.Data {
		if data.Type == "primary" {
			primaryHref = data.Location.Href
			break
		}
	}
	if primaryHref == "" {
		errorf("no primary in repo metadata")
		return 1
	}
	primaryRelURL, err := url.Parse(primaryHref)
	if err != nil {
		errorf("failed to parse primary location from repo metadata: %s", err)
		return 1
	}
	primaryURL := repoURL.ResolveReference(primaryRelURL)
	debugf("downloading primary metadata from %s", primaryURL)
	response, err = get(primaryURL)
	if err != nil {
		errorf("failed to download primary metadata: %s", err)
		return 1
	}
	var primary yum.Primary
	if strings.HasSuffix(primaryURL.Path, ".gz") {
		debugf("using gzip to decompress primary metadata")
		reader, err := gzip.NewReader(response.Body)
		if err != nil {
			errorf("failed to decompress gzipped primary metadata: %s", err)
			return 1
		}
		decoder = xml.NewDecoder(reader)
	} else if strings.HasSuffix(primaryURL.Path, ".bz2") {
		debugf("using bzip2 to decompress primary metadata")
		decoder = xml.NewDecoder(bzip2.NewReader(response.Body))
	} else {
		decoder = xml.NewDecoder(response.Body)
	}
	if err := decoder.Decode(&primary); err != nil {
		errorf("failed to deode primary metadata as XML: %s", err)
		return 1
	}
	debugf("primary metadata lists %d packages", len(primary.PackageList))
	if *flagList {
		debugf("listing packages available in repo")
		for _, pkg := range primary.PackageList {
			fmt.Printf("%s-%s-%s (%s): %s\n", pkg.Name, pkg.Version.Ver, pkg.Version.Rel, pkg.Arch, pkg.Summary)
		}
		return 0
	}
	for _, pkgToGet := range pkgsToGet {
		pkgParts := rePkgName.FindStringSubmatch(pkgToGet)
		if len(pkgParts) != 4 {
			errorf("must specify package in name-ver-rel format, e.g. foobar-1.2.3-4")
			return 1
		}
		pkgName, pkgVer, pkgRel := pkgParts[1], pkgParts[2], pkgParts[3]
		if pkgName == "" || pkgVer == "" || pkgRel == "" {
			errorf("must specify name, ver, and rel parameters of package as nonempty strings")
			return 1
		}
		debugf("searching for package name %s, ver %s, rel %s", pkgName, pkgVer, pkgRel)
		var lastPkg yum.Package
		for _, pkg := range primary.PackageList {
			if pkg.Name == pkgName && pkg.Version.Ver == pkgVer && pkg.Version.Rel == pkgRel {
				if lastPkg.Name != "" {
					if lastPkg.Version.Epoch < pkg.Version.Epoch {
						lastPkg = pkg
					}
				} else {
					lastPkg = pkg
				}
			}
		}
		if lastPkg.Name == "" {
			errorf("failed to find package %s in repository", pkgToGet)
			return 1
		}
		debugf("using epoch %d", lastPkg.Version.Epoch)
		pkgRelURL, err := url.Parse(lastPkg.Location.Href)
		if err != nil {
			errorf("failed to parse package location: %s", err)
			return 1
		}
		pkgURL := repoURL.ResolveReference(pkgRelURL)
		debugf("downloading package from %s", pkgURL)
		response, err = get(pkgURL)
		if err != nil {
			errorf("failed to download package: %s", err)
			return 1
		}
		fileName := path.Base(pkgURL.Path)
		file, err := os.Create(fileName)
		if err != nil {
			errorf("failed to open output file: %s", err)
			return 1
		}
		if n, err := io.Copy(file, response.Body); err != nil {
			errorf("failed to copy package to output file: %s", err)
			return 1
		} else {
			debugf("downloaded %d bytes", n)
		}
		fmt.Println(fileName)
	}
	return 0
}

func debugf(format string, args ...interface{}) {
	if *flagVerbose {
		fmt.Fprintf(os.Stderr, "DEBUG %s\n", fmt.Sprintf(format, args...))
	}
}

func errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR %s\n", fmt.Sprintf(format, args...))
}

func get(requestURL *url.URL) (*http.Response, error) {
	request := &http.Request{
		URL:    requestURL,
		Method: http.MethodGet,
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	} else if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("unexpected status %s", response.Status)
	}
	return response, nil
}

func mustParseURL(raw string) *url.URL {
	parsed, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return parsed
}
