package bitmarksdk

import (
	"encoding/hex"
	"io/ioutil"
	"path/filepath"
	"reflect"

	"golang.org/x/crypto/sha3"
)

type Accessibility string

const (
	Public  Accessibility = "public"
	Private Accessibility = "private"
)

type AssetInfo struct {
	Name     string
	Metadata map[string]string
}

type AssetFile struct {
	propertyName     string
	propertyMetadata map[string]string

	Path          string
	Name          string
	Content       []byte
	Fingerprint   string
	Accessibility Accessibility
}

func NewAssetFileFromPath(path string, acs Accessibility) (*AssetFile, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	digest := sha3.Sum512(content)
	fingerprint := "01" + hex.EncodeToString(digest[:])

	return &AssetFile{
		Path:          path,
		Name:          filepath.Base(path),
		Content:       content,
		Fingerprint:   fingerprint,
		Accessibility: acs,
	}, nil
}

func NewAssetFile(name string, content []byte, acs Accessibility) *AssetFile {
	digest := sha3.Sum512(content)
	fingerprint := "01" + hex.EncodeToString(digest[:])

	return &AssetFile{
		Name:          name,
		Content:       content,
		Fingerprint:   fingerprint,
		Accessibility: acs,
	}
}

func (af *AssetFile) Id() string {
	assetIndex := sha3.Sum512([]byte(af.Fingerprint))
	return hex.EncodeToString(assetIndex[:])
}

func (af *AssetFile) equivalent(asset *Asset) bool {
	if af.propertyName == "" {
		return true
	}
	return af.propertyName == asset.Name && reflect.DeepEqual(af.propertyMetadata, asset.Metadata)
}
