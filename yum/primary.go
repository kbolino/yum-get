package yum

type Primary struct {
	PackageList []Package `xml:"package"`
}

type Package struct {
	Name     string   `xml:"name"`
	Arch     string   `xml:"arch"`
	Version  Version  `xml:"version"`
	Checksum Checksum `xml:"checksum"`
	Location Location `xml:"location"`
	Summary  string   `xml:"summary"`
}

type Version struct {
	Ver   string `xml:"ver,attr"`
	Rel   string `xml:"rel,attr"`
	Epoch int64  `xml:"epoch,attr"`
}
