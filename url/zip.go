package url

import (
	"archive/zip"
	"fmt"
	"io"
	"path"
	"strings"
)

// Note: we *must* use the "path" package rather than "filepath" to ensure consistenty with Windows

//
// ZipURL
//

type ZipURL struct {
	Path       string
	ArchiveURL *FileURL
}

func NewZipURL(path string, archiveUrl *FileURL) *ZipURL {
	return &ZipURL{path, archiveUrl}
}

func NewZipURLFromURL(url string) (*ZipURL, error) {
	archive, path, err := ParseZipURL(url)
	if err != nil {
		return nil, err
	}

	archiveUrl, err := NewURL(path)
	if err != nil {
		return nil, err
	}

	return NewZipURL(archive, archiveUrl.(*FileURL)), nil
}

func NewValidZipURL(path string, archiveURL *FileURL) (*ZipURL, error) {
	archiveReader, err := zip.OpenReader(archiveURL.Path)
	if err != nil {
		return nil, err
	}
	defer archiveReader.Close()

	if (len(path) > 0) && strings.HasPrefix(path, "/") {
		// Must be absolute
		path = path[1:]
	}

	if strings.Contains(path, "TOSCA-Metadata/TOSCA.meta") {
		for _, file := range archiveReader.File {
			if strings.Contains(file.Name, "TOSCA-Metadata/TOSCA.meta") {
				path = file.Name
			}
		}
	}

	for _, file := range archiveReader.File {
		if path == file.Name {
			return NewZipURL(path, archiveURL), nil
		}
	}

	return nil, fmt.Errorf("path not found in zip: %s", path)
}

func NewValidZipURLFromURL(url string) (*ZipURL, error) {
	archive, path, err := ParseZipURL(url)
	if err != nil {
		return nil, err
	}

	archiveUrl, err := NewValidURL(archive, nil)
	if err != nil {
		return nil, err
	}

	switch u := archiveUrl.(type) {
	case *FileURL:
		return NewValidZipURL(path, u)
		// TODO": download NetURL?
	}

	return nil, fmt.Errorf("unsupported archive URL type in \"zip:\" URL: %s", url)
}

func NewValidRelativeZipURL(path_ string, origin *ZipURL) (*ZipURL, error) {
	path_ = path.Join(origin.Path, path_)
	return NewValidZipURL(path_, origin.ArchiveURL)
}

func (self *ZipURL) OpenArchive() (*zip.ReadCloser, error) {
	return zip.OpenReader(self.ArchiveURL.Path)
}

// URL interface
func (self *ZipURL) String() string {
	return self.Key()
}

// URL interface
func (self *ZipURL) Format() string {
	return GetFormat(self.Path)
}

// URL interface
func (self *ZipURL) Origin() URL {
	return &ZipURL{path.Dir(self.Path), self.ArchiveURL}
}

// URL interface
func (self *ZipURL) Key() string {
	return fmt.Sprintf("zip:%s!/%s", self.ArchiveURL.String(), self.Path)
}

// URL interface
func (self *ZipURL) Open() (io.Reader, error) {
	archiveReader, err := self.OpenArchive()
	if err != nil {
		return nil, err
	}

	for _, file := range archiveReader.File {
		if self.Path == file.Name {
			fileReader, err := file.Open()
			if err != nil {
				archiveReader.Close()
				return nil, err
			}
			return ZipFileReadCloser{fileReader, archiveReader}, nil
		}
	}

	archiveReader.Close()
	return nil, fmt.Errorf("path not found in zip: %s", self.Path)
}

func ParseZipURL(url string) (string, string, error) {
	if !strings.HasPrefix(url, "zip:") {
		return "", "", fmt.Errorf("not a \"zip:\" URL: %s", url)
	}

	split := strings.Split(url[4:], "!")
	if len(split) != 2 {
		return "", "", fmt.Errorf("malformed \"zip:\" URL: %s", url)
	}

	return split[0], split[1], nil
}

//
// ZipFileReadCloser
//

type ZipFileReadCloser struct {
	FileReader    io.ReadCloser
	ArchiveReader *zip.ReadCloser
}

// io.Reader interface
func (self ZipFileReadCloser) Read(p []byte) (n int, err error) {
	return self.FileReader.Read(p)
}

// io.Closer interface
func (self ZipFileReadCloser) Close() error {
	self.FileReader.Close()
	return self.ArchiveReader.Close()
}
