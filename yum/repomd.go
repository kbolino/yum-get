package yum

type RepoMD struct {
	Revision string     `xml:"revision"`
	Data     []FileData `xml:"data"`
}

type FileData struct {
	Type      string   `xml:"type,attr"`
	Checksum  Checksum `xml:"checksum"`
	Location  Location `xml:"location"`
	Timestamp int64    `xml:"timestamp"`
}

type Checksum struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",innerxml"`
}

type Location struct {
	Href string `xml:"href,attr"`
}
